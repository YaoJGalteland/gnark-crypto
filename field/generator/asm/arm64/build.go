package arm64

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/consensys/bavard/arm64"
)

type defineFn func(args ...arm64.Register)

func NewFFArm64(w io.Writer, nbWords int) *FFArm64 {
	F := &FFArm64{
		arm64.NewArm64(w),
		0,
		0,
		nbWords,
		nbWords - 1,
		make([]int, nbWords),
		make([]int, nbWords-1),
		make(map[string]defineFn),
	}

	// indexes (template helpers)
	for i := 0; i < F.NbWords; i++ {
		F.NbWordsIndexesFull[i] = i
		if i > 0 {
			F.NbWordsIndexesNoZero[i-1] = i
		}
	}

	return F
}

type FFArm64 struct {
	*arm64.Arm64
	nbElementsOnStack    int
	maxOnStack           int
	NbWords              int
	NbWordsLastIndex     int
	NbWordsIndexesFull   []int
	NbWordsIndexesNoZero []int
	mDefines             map[string]defineFn
}

// GenerateCommonASM generates assembly code for the base field provided to goff
// see internal/templates/ops*
func GenerateCommonASM(w io.Writer, nbWords, nbBits int, hasVector bool) error {
	f := NewFFArm64(w, nbWords)
	f.Comment("Code generated by gnark-crypto/generator. DO NOT EDIT.")

	f.WriteLn("#include \"textflag.h\"")
	f.WriteLn("#include \"funcdata.h\"")
	f.WriteLn("#include \"go_asm.h\"")
	f.WriteLn("")

	if nbWords == 1 {
		if nbBits <= 31 {
			return GenerateF31ASM(f, hasVector)
		} else {
			panic("not implemented")
		}
	}

	if f.NbWords%2 != 0 {
		panic("NbWords must be even")
	}

	if f.NbWords <= 6 {
		f.generateButterfly()
	}
	f.generateMul()
	f.generateReduce()

	return nil
}

func (f *FFArm64) DefineFn(name string) (fn defineFn, err error) {
	fn, ok := f.mDefines[name]
	if !ok {
		return nil, fmt.Errorf("function %s not defined", name)
	}
	return fn, nil
}

func (f *FFArm64) Define(name string, nbInputs int, fn defineFn) defineFn {

	inputs := make([]string, nbInputs)
	for i := 0; i < nbInputs; i++ {
		inputs[i] = fmt.Sprintf("in%d", i)
	}
	name = strings.ToUpper(name)

	for _, ok := f.mDefines[name]; ok; {
		// name already exist, for code generation purpose we add a suffix
		// should happen only with e2 deprecated functions
		// fmt.Println("WARNING: function name already defined, adding suffix")
		i := 0
		for {
			newName := fmt.Sprintf("%s_%d", name, i)
			if _, ok := f.mDefines[newName]; !ok {
				name = newName
				goto startDefine
			}
			i++
		}
	}
startDefine:

	f.StartDefine()
	f.WriteLn("#define " + name + "(" + strings.Join(inputs, ", ") + ")")
	inputsRegisters := make([]arm64.Register, nbInputs)
	for i := 0; i < nbInputs; i++ {
		inputsRegisters[i] = arm64.Register(inputs[i])
	}
	fn(inputsRegisters...)
	f.EndDefine()
	f.WriteLn("")

	toReturn := func(args ...arm64.Register) {
		if len(args) != nbInputs {
			panic("invalid number of arguments")
		}
		inputsStr := make([]string, len(args))
		for i := 0; i < len(args); i++ {
			inputsStr[i] = string(args[i])
		}
		f.WriteLn(name + "(" + strings.Join(inputsStr, ", ") + ")")
	}

	f.mDefines[name] = toReturn

	return toReturn
}

func (f *FFArm64) AssertCleanStack(reservedStackSize, minStackSize int) {
	if f.nbElementsOnStack != 0 {
		panic("missing f.Push stack elements")
	}
	if reservedStackSize < minStackSize {
		panic("invalid minStackSize or reservedStackSize")
	}
	usedStackSize := f.maxOnStack * 8
	if usedStackSize > reservedStackSize {
		panic("using more stack size than reserved")
	} else if max(usedStackSize, minStackSize) < reservedStackSize {
		// this panic is for dev purposes as this may be by design for aligment
		panic("reserved more stack size than needed")
	}

	f.maxOnStack = 0
}

func (f *FFArm64) Push(registers *arm64.Registers, rIn ...arm64.Register) {
	for _, r := range rIn {
		if strings.HasPrefix(string(r), "s") {
			// it's on the stack, decrease the offset
			f.nbElementsOnStack--
			continue
		}
		registers.Push(r)
	}
}

func (f *FFArm64) Pop(registers *arm64.Registers, forceStack ...bool) arm64.Register {
	if registers.Available() >= 1 && !(len(forceStack) > 0 && forceStack[0]) {
		return registers.Pop()
	}
	r := arm64.Register(fmt.Sprintf("s%d-%d(SP)", f.nbElementsOnStack, 8+f.nbElementsOnStack*8))
	f.nbElementsOnStack++
	if f.nbElementsOnStack > f.maxOnStack {
		f.maxOnStack = f.nbElementsOnStack
	}
	return r
}

func (f *FFArm64) PopN(registers *arm64.Registers, forceStack ...bool) []arm64.Register {
	if len(forceStack) > 0 && forceStack[0] {
		nbStack := f.NbWords
		var u []arm64.Register

		for i := f.nbElementsOnStack; i < nbStack+f.nbElementsOnStack; i++ {
			u = append(u, arm64.Register(fmt.Sprintf("s%d-%d(SP)", i, 8+i*8)))
		}
		f.nbElementsOnStack += nbStack
		if f.nbElementsOnStack > f.maxOnStack {
			f.maxOnStack = f.nbElementsOnStack
		}
		return u
	}
	if registers.Available() >= f.NbWords {
		return registers.PopN(f.NbWords)
	}
	nbStack := f.NbWords - registers.Available()
	u := registers.PopN(registers.Available())

	for i := f.nbElementsOnStack; i < nbStack+f.nbElementsOnStack; i++ {
		u = append(u, arm64.Register(fmt.Sprintf("s%d-%d(SP)", i, 8+i*8)))
	}
	f.nbElementsOnStack += nbStack
	if f.nbElementsOnStack > f.maxOnStack {
		f.maxOnStack = f.nbElementsOnStack
	}
	return u
}

func (f *FFArm64) qAt(index int) string {
	return fmt.Sprintf("·qElement+%d(SB)", index*8)
}

func (f *FFArm64) qInv0() string {
	return "$const_qInvNeg"
}

func ElementASMFileName(nbWords, nbBits int) string {
	const nameW1 = "element_%db_arm64.s"
	const nameWN = "element_%dw_arm64.s"
	const fW1 = "element_%db"
	const fWN = "element_%dw"
	if nbWords == 1 {
		return filepath.Join(fmt.Sprintf(fW1, nbBits), fmt.Sprintf(nameW1, nbBits))
	}
	return filepath.Join(fmt.Sprintf(fWN, nbWords), fmt.Sprintf(nameWN, nbWords))
}

func GenerateF31ASM(f *FFArm64, hasVector bool) error {
	if !hasVector {
		return nil // nothing for now.
	}

	f.generateAddVecF31()
	f.generateSubVecF31()
	f.generateSumVecF31()

	return nil
}
