package tfhe

type TGswParams struct {
	l          int32       ///< decomp length
	Bgbit      int32       ///< log_2(Bg)
	Bg         int32       ///< decomposition base (must be a power of 2)
	halfBg     int32       ///< Bg/2
	maskMod    uint32      ///< Bg-1
	tlweParams *TLweParams ///< Params of each row
	kpl        int32       ///< number of rows = (k+1)*l
	H          []Torus32   ///< powers of Bgbit
	offset     uint32      ///< offset = Bg/2 * (2^(32-Bgbit) + 2^(32-2*Bgbit) + ... + 2^(32-l*Bgbit))
}

type TGswKey struct {
	params     *TGswParams     ///< the parameters
	tlweParams *TLweParams     ///< the tlwe params of each rows
	key        []IntPolynomial ///< the key (array of k polynomials)
	TlweKey    TLweKey
}

type TGswSample struct {
	AllSample  []TLweSample   ///< TLweSample* all_sample; (k+1)l TLwe Sample
	BlocSample [][]TLweSample ///< optional access to the different size l blocks
	// double current_variance
	k int32
	l int32
}

/*

func TGswClear(result *TGswSample, params *TGswParams) {
	kpl := params.kpl
	for p := int32(0); p < kpl; p++ {
		tLweClear(&result.AllSample[p], params.tlweParams)
	}
}

// Result += H
func TGswAddH(result *TGswSample, params *TGswParams) {
	k := params.tlweParams.k
	l := params.l
	h := params.H
	// compute result += H
	for bloc := int32(0); bloc <= k; bloc++ {
		for i := int32(0); i < l; i++ {
			result.BlocSample[bloc][i].A[bloc].CoefsT[0] += h[i]
		}
	}
}
*/

func NewTGswSample(params *TGswParams) *TGswSample {
	k := params.tlweParams.k
	l := params.l
	kpl := params.kpl
	//h := params.H

	allSamples := make([]TLweSample, kpl)
	for i := range allSamples {
		allSamples[i] = *NewTLweSample(params.tlweParams)
	}
	blocSamples := make([][]TLweSample, k+1)
	for i := range blocSamples {
		blocSamples[i] = make([]TLweSample, l)
		for j := range blocSamples[i] {
			blocSamples[i][j] = *NewTLweSample(params.tlweParams)
		}
	}
	return &TGswSample{
		AllSample:  allSamples,
		BlocSample: blocSamples,
		k:          k,
		l:          l,
	}
}

func NewTGswSample2(AllSample []TLweSample, BlocSample [][]TLweSample, k, l int32) *TGswSample {
	return &TGswSample{AllSample, BlocSample, k, l}
}

func NewTGswParams(l, Bgbit int32, tlwe_params *TLweParams) *TGswParams {
	var Bg int32 = 1 << Bgbit
	var halfBg int32 = Bg / 2
	h := make([]Torus32, l)
	for i := int32(0); i < l; i++ {
		kk := (32 - (i+1)*Bgbit)
		h[i] = 1 << kk // 1/(Bg^(i+1)) as a Torus32
	}

	// offset = Bg/2 * (2^(32-Bgbit) + 2^(32-2*Bgbit) + ... + 2^(32-l*Bgbit))
	var temp1 int32 = 0
	for i := int32(0); i < l; i++ {
		temp0 := int32(1 << (32 - (i+1)*Bgbit))
		temp1 += temp0
	}
	offset := temp1 * halfBg

	return &TGswParams{
		Bg:         Bg,
		l:          l,
		Bgbit:      Bgbit,
		halfBg:     halfBg,
		maskMod:    uint32(Bg - 1),
		tlweParams: tlwe_params,
		kpl:        int32((tlwe_params.k + 1) * l),
		H:          h,
		offset:     uint32(offset),
	}
}

func NewTGswKey(params *TGswParams) *TGswKey {
	tlweKey := *NewTLweKey(params.tlweParams)
	return &TGswKey{
		params:     params,
		tlweParams: params.tlweParams,
		TlweKey:    tlweKey,
		key:        tlweKey.key,
	}
}

/*
func init_TGswSample(obj *TGswSample, params *TGswParams) {
	k := params.tlweParams.k
	l := params.l
	all_sample := NewTLweSampleArray((k+1)*l, params.tlweParams) // all samples as a line vector
	//TLweSample **bloc_sample = new TLweSample *[k + 1]; // horizontal blocks (l rows) of the TGsw matrix

	bloc_sample := make([][]TLweSample, k+1)

	for p := int32(0); p < k+1; p++ {
		bloc_sample[p] = all_sample[p*l] // all_sample + p * l
	}
	obj = NewTGswSample(all_sample, bloc_sample, k, l)
}
*/

// TGsw
/** generate a tgsw key (in fact, a tlwe key) */
func TGswKeyGen(result *TGswKey) {
	tLweKeyGen(&result.TlweKey)
}

// support Functions for TGsw
// Result = 0
func TGswClear(result *TGswSample, params *TGswParams) {
	kpl := params.kpl
	for p := int32(0); p < kpl; p++ {
		tLweClear(&result.AllSample[p], params.tlweParams)
	}
}

// Result += H
func TGswAddH(result *TGswSample, params *TGswParams) {
	k := params.tlweParams.k
	l := params.l
	h := params.H
	// compute result += H
	for bloc := int32(0); bloc <= k; bloc++ {
		for i := int32(0); i < l; i++ {
			result.BlocSample[bloc][i].A[bloc].CoefsT[0] += h[i]
		}
	}
}

// Result += mu*H
func TGswAddMuH(result *TGswSample, message *IntPolynomial, params *TGswParams) {
	k := params.tlweParams.k
	N := params.tlweParams.N
	l := params.l
	h := params.H
	mu := message.Coefs

	// compute result += H
	for bloc := int32(0); bloc <= k; bloc++ {
		for i := int32(0); i < l; i++ {
			target := result.BlocSample[bloc][i].A[bloc].CoefsT
			hi := h[i]
			for j := int32(0); j < N; j++ {
				target[j] += mu[j] * hi
			}
		}
	}
}

// Result += mu*H, mu integer
func TGswAddMuIntH(result *TGswSample, message int32, params *TGswParams) {
	k := params.tlweParams.k
	l := params.l
	h := params.H

	// compute result += H
	for bloc := int32(0); bloc <= k; bloc++ {
		for i := int32(0); i < l; i++ {
			result.BlocSample[bloc][i].A[bloc].CoefsT[0] += message * h[i]
		}
	}
}

// Result = tGsw(0)
func TGswEncryptZero(result *TGswSample, alpha double, key *TGswKey) {
	rlkey := &key.TlweKey
	kpl := key.params.kpl
	for p := int32(0); p < kpl; p++ {
		tLweSymEncryptZero(&result.AllSample[p], alpha, rlkey)
	}
}

//mult externe de X^{a_i} par bki
func TGswMulByXaiMinusOne(result *TGswSample, ai int32, bk *TGswSample, params *TGswParams) {
	par := params.tlweParams
	kpl := params.kpl
	for i := int32(0); i < kpl; i++ {
		tLweMulByXaiMinusOne(&result.AllSample[i], ai, &bk.AllSample[i], par)
	}
}

//Update l'accumulateur ligne 5 de l'algo toujours
//void tGswTLweDecompH(IntPolynomial* result, const TLweSample* sample,const TGswParams* params)
//accum *= sample
func TGswExternMulToTLwe(accum *TLweSample, sample *TGswSample, params *TGswParams) {
	par := params.tlweParams
	N := par.N
	kpl := int(params.kpl)
	//TODO: improve this new/delete
	dec := NewIntPolynomialArray(kpl, N)

	TGswTLweDecompH(dec, accum, params)
	tLweClear(accum, par)
	for i := 0; i < kpl; i++ {
		tLweAddMulRTo(accum, &dec[i], &sample.AllSample[i], par)
	}
}

/**
 * encrypts a poly message
 */
func TGswSymEncrypt(result *TGswSample, message *IntPolynomial, alpha double, key *TGswKey) {
	TGswEncryptZero(result, alpha, key)
	TGswAddMuH(result, message, key.params)
}

/**
 * encrypts a constant message
 */
func TGswSymEncryptInt(result *TGswSample, message int32, alpha double, key *TGswKey) {
	TGswEncryptZero(result, alpha, key)
	TGswAddMuIntH(result, message, key.params)
}

/**
 * encrypts a message = 0 ou 1
 */
func TGswEncryptB(result *TGswSample, message int32, alpha double, key *TGswKey) {
	TGswEncryptZero(result, alpha, key)
	if message == 1 {
		TGswAddH(result, key.params)
	}
}

// à revoir
func TGswSymDecrypt(result *IntPolynomial, sample *TGswSample, key *TGswKey, Msize int) {
	params := key.params
	rlwe_params := params.tlweParams
	N := rlwe_params.N
	l := params.l
	k := rlwe_params.k
	testvec := NewTorusPolynomial(N)
	tmp := NewTorusPolynomial(N)
	decomp := NewIntPolynomialArray(int(l), N)

	indic := ModSwitchToTorus32(1, int32(Msize))
	torusPolynomialClear(testvec)
	testvec.CoefsT[0] = indic
	TGswTorus32PolynomialDecompH(decomp, testvec, params)

	torusPolynomialClear(testvec)
	for i := int32(0); i < l; i++ {
		for j := int32(1); j < N; j++ {
			Assert(decomp[i].Coefs[j] == 0)
		}
		TLwePhase(tmp, &sample.BlocSample[k][i], &key.TlweKey)
		TorusPolynomialAddMulR(testvec, &decomp[i], tmp)
	}
	for i := int32(0); i < N; i++ {
		result.Coefs[i] = ModSwitchFromTorus32(testvec.CoefsT[i], Msize)
	}
}

/*
// à revoir
EXPORT int32_t tGswSymDecryptInt(const TGswSample* sample, const TGswKey* key){
    TorusPolynomial* phase = new_TorusPolynomial(key.params.tlwe_params.N)

    tGswPhase(phase, sample, key)
    int32_t result = modSwitchFromTorus32(phase.CoefsT[0], Msize)

    delete_TorusPolynomial(phase)
    return result
}
*/
//do we really decrypt Gsw samples?
// EXPORT void tGswMulByXaiMinusOne(Gsw* result, int32_t ai, const Gsw* bk)
// EXPORT void tLweExternMulRLweTo(RLwe* accum, Gsw* a); //  accum = a \odot accum

//fonction de decomposition
func TGswTLweDecompH(result []IntPolynomial, sample *TLweSample, params *TGswParams) {
	k := params.tlweParams.k
	l := params.l
	/*
		for i := int32(0); i < k; i++ { // b=a[k]

			//tGswTorus32PolynomialDecompH(result+(i*l), &sample.a[i], params)

			// sort of works only when i < k
			TGswTorus32PolynomialDecompH(result[i:i+l], &sample.A[i], params)

			//TGswTorus32PolynomialDecompH(result[i:i+l], &sample.A[i], params)

			//TGswTorus32PolynomialDecompH(result[l:], &sample.A[i], params)
			//TGswTorus32PolynomialDecompH(result, &sample.A[i], params)
			//TGswTorus32PolynomialDecompH(result[:i+l], &sample.A[i], params)
		}
	*/

	var j = 0
	for i := int32(0); i <= k*l; i += l {

		/*
			sub := result[i : i+l]
			fmt.Printf("len(sub): %d\n", len(sub))
			for _, v := range sub {
				fmt.Println(v.Coefs[0])
			}
		*/

		TGswTorus32PolynomialDecompH(result[i:i+l], &sample.A[j], params)
		j++
	}

}

func Torus32PolynomialDecompH_old(result []IntPolynomial, sample *TorusPolynomial, params *TGswParams) {
	N := params.tlweParams.N
	l := params.l
	Bgbit := params.Bgbit
	maskMod := params.maskMod
	halfBg := params.halfBg
	offset := params.offset

	for j := int32(0); j < N; j++ {
		temp0 := uint32(sample.CoefsT[j]) + offset
		for p := int32(0); p < l; p++ {
			temp1 := (temp0 >> (32 - (p+1)*Bgbit)) & maskMod // doute
			result[p].Coefs[j] = int32(temp1) - halfBg
		}
	}
}

func TGswTorus32PolynomialDecompH(result []IntPolynomial, sample *TorusPolynomial, params *TGswParams) {
	N := params.tlweParams.N
	l := params.l
	Bgbit := params.Bgbit
	buf := []uint32{}
	for _, vNum := range sample.CoefsT {
		buf = append(buf, uint32(vNum))
	}
	maskMod := params.maskMod
	halfBg := params.halfBg
	offset := params.offset
	//First, add offset to everyone
	for j := int32(0); j < N; j++ {
		buf[j] += offset
		//sample.CoefsT[j] += Torus32(offset)
	}

	//then, do the decomposition (in parallel)
	for p := int32(0); p < l; p++ {
		var decal int32 = 32 - (p+1)*Bgbit
		//res_p := result[p].Coefs
		for j := int32(0); j < N; j++ {
			var temp1 int32 = int32((buf[j] >> uint32(decal)) & maskMod)
			//var temp1 int32 = int32((uint32(sample.CoefsT[j]) >> uint32(decal)) & maskMod)
			result[p].Coefs[j] = temp1 - halfBg
		}
	}
	//finally, remove offset from everyone
	for j := int32(0); j < N; j++ {
		buf[j] -= offset
		//sample.CoefsT[j] -= Torus32(offset)
	}
}

//result = a*b
func TGswExternProduct(result *TLweSample, a *TGswSample, b *TLweSample, params *TGswParams) {
	parlwe := params.tlweParams
	N := parlwe.N
	kpl := params.kpl
	dec := NewIntPolynomialArray(int(kpl), N)
	TGswTLweDecompH(dec, b, params)
	tLweClear(result, parlwe)
	for i := int32(0); i < kpl; i++ {
		tLweAddMulRTo(result, &dec[i], &a.AllSample[i], parlwe)
	}
	result.CurrentVariance += b.CurrentVariance //todo + the error term?
}

/**
 * result = (0,mu)
 */
func TGswNoiselessTrivial(result *TGswSample, mu *IntPolynomial, params *TGswParams) {
	TGswClear(result, params)
	TGswAddMuH(result, mu, params)
}
