package runtime

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"fmt"
	"io"
	goruntime "runtime"
)

// validateExecutableHeader verifies only the host binary family and CPU. It
// is not a malware, provenance, or general executable-safety assessment.
func validateExecutableHeader(reader io.ReaderAt) error {
	switch goruntime.GOOS {
	case "windows":
		file, err := pe.NewFile(reader)
		if err != nil {
			return invalidExecutableHeader("PE", err)
		}
		defer file.Close()
		if file.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 {
			return fmt.Errorf(
				"%w: PE image is not marked executable",
				ErrMaterializedExecutableInvalid,
			)
		}
		return validateHostPEMachine(file.Machine)
	case "linux":
		file, err := elf.NewFile(reader)
		if err != nil {
			return invalidExecutableHeader("ELF", err)
		}
		defer file.Close()
		if file.Type != elf.ET_EXEC && file.Type != elf.ET_DYN {
			return fmt.Errorf(
				"%w: ELF type %s is not executable",
				ErrMaterializedExecutableInvalid,
				file.Type,
			)
		}
		return validateHostELFMachine(file.Machine)
	case "darwin":
		file, err := macho.NewFile(reader)
		if err != nil {
			return invalidExecutableHeader("Mach-O", err)
		}
		defer file.Close()
		if file.Type != macho.TypeExec {
			return fmt.Errorf(
				"%w: Mach-O type %s is not executable",
				ErrMaterializedExecutableInvalid,
				file.Type,
			)
		}
		return validateHostMachOCPU(file.Cpu)
	default:
		return fmt.Errorf(
			"%w: executable header validation does not support host OS %q",
			ErrMaterializedExecutableInvalid,
			goruntime.GOOS,
		)
	}
}

func validateHostPEMachine(machine uint16) error {
	want, ok := hostPEMachine(goruntime.GOARCH)
	if !ok {
		return unsupportedHeaderArchitecture("PE", goruntime.GOARCH)
	}
	if machine != want {
		return fmt.Errorf(
			"%w: PE machine %#x does not match host architecture %s",
			ErrMaterializedExecutableInvalid,
			machine,
			goruntime.GOARCH,
		)
	}
	return nil
}

func hostPEMachine(architecture string) (uint16, bool) {
	switch architecture {
	case "386":
		return pe.IMAGE_FILE_MACHINE_I386, true
	case "amd64":
		return pe.IMAGE_FILE_MACHINE_AMD64, true
	case "arm":
		return pe.IMAGE_FILE_MACHINE_ARMNT, true
	case "arm64":
		return pe.IMAGE_FILE_MACHINE_ARM64, true
	default:
		return 0, false
	}
}

func validateHostELFMachine(machine elf.Machine) error {
	want, ok := hostELFMachine(goruntime.GOARCH)
	if !ok {
		return unsupportedHeaderArchitecture("ELF", goruntime.GOARCH)
	}
	if machine != want {
		return fmt.Errorf(
			"%w: ELF machine %s does not match host architecture %s",
			ErrMaterializedExecutableInvalid,
			machine,
			goruntime.GOARCH,
		)
	}
	return nil
}

func hostELFMachine(architecture string) (elf.Machine, bool) {
	switch architecture {
	case "386":
		return elf.EM_386, true
	case "amd64":
		return elf.EM_X86_64, true
	case "arm":
		return elf.EM_ARM, true
	case "arm64":
		return elf.EM_AARCH64, true
	case "mips", "mipsle", "mips64", "mips64le":
		return elf.EM_MIPS, true
	case "ppc64", "ppc64le":
		return elf.EM_PPC64, true
	case "riscv64":
		return elf.EM_RISCV, true
	case "s390x":
		return elf.EM_S390, true
	case "loong64":
		return elf.EM_LOONGARCH, true
	default:
		return elf.EM_NONE, false
	}
}

func validateHostMachOCPU(cpu macho.Cpu) error {
	want, ok := hostMachOCPU(goruntime.GOARCH)
	if !ok {
		return unsupportedHeaderArchitecture("Mach-O", goruntime.GOARCH)
	}
	if cpu != want {
		return fmt.Errorf(
			"%w: Mach-O CPU %s does not match host architecture %s",
			ErrMaterializedExecutableInvalid,
			cpu,
			goruntime.GOARCH,
		)
	}
	return nil
}

func hostMachOCPU(architecture string) (macho.Cpu, bool) {
	switch architecture {
	case "386":
		return macho.Cpu386, true
	case "amd64":
		return macho.CpuAmd64, true
	case "arm":
		return macho.CpuArm, true
	case "arm64":
		return macho.CpuArm64, true
	default:
		return 0, false
	}
}

func invalidExecutableHeader(family string, err error) error {
	return fmt.Errorf(
		"%w: parse host %s executable header: %w",
		ErrMaterializedExecutableInvalid,
		family,
		err,
	)
}

func unsupportedHeaderArchitecture(family, architecture string) error {
	return fmt.Errorf(
		"%w: %s header validation does not support host architecture %q",
		ErrMaterializedExecutableInvalid,
		family,
		architecture,
	)
}
