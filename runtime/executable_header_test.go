package runtime

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"errors"
	goruntime "runtime"
	"testing"
)

func TestExecutableHeaderAcceptsRealHostExecutable(t *testing.T) {
	loaded := loadProcessRunnerFixture(t, "process_success")
	executable := loaded.ExecutableBytes()
	if len(executable) == 0 {
		t.Fatal("real host executable is empty")
	}
	if err := validateExecutableHeader(bytes.NewReader(executable)); err != nil {
		t.Fatalf("real host executable header rejected: %v", err)
	}
}

func TestExecutableHeaderRejectsNonExecutableInputs(t *testing.T) {
	tests := map[string][]byte{
		"plain text": []byte("not an executable"),
		"script":     []byte("#!/bin/sh\necho unsafe\n"),
		"truncated":  {0x7f, 'E', 'L', 'F'},
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateExecutableHeader(bytes.NewReader(input))
			if !errors.Is(err, ErrMaterializedExecutableInvalid) {
				t.Fatalf("expected ErrMaterializedExecutableInvalid, got %v", err)
			}
		})
	}
}

func TestExecutableHeaderRejectsWrongArchitecture(t *testing.T) {
	var err error
	switch goruntime.GOOS {
	case "windows":
		wrong := uint16(pe.IMAGE_FILE_MACHINE_I386)
		if goruntime.GOARCH == "386" {
			wrong = pe.IMAGE_FILE_MACHINE_AMD64
		}
		err = validateHostPEMachine(wrong)
	case "linux":
		wrong := elf.EM_386
		if goruntime.GOARCH == "386" {
			wrong = elf.EM_X86_64
		}
		err = validateHostELFMachine(wrong)
	case "darwin":
		wrong := macho.Cpu386
		if goruntime.GOARCH == "386" {
			wrong = macho.CpuAmd64
		}
		err = validateHostMachOCPU(wrong)
	default:
		t.Skipf("host executable family is not supported on %s", goruntime.GOOS)
	}
	if !errors.Is(err, ErrMaterializedExecutableInvalid) {
		t.Fatalf("expected ErrMaterializedExecutableInvalid, got %v", err)
	}
}

func TestExecutableHeaderArchitectureMappingsRejectUnknownArchitecture(t *testing.T) {
	if _, ok := hostPEMachine("forge-unknown"); ok {
		t.Fatal("PE mapping accepted an unknown architecture")
	}
	if _, ok := hostELFMachine("forge-unknown"); ok {
		t.Fatal("ELF mapping accepted an unknown architecture")
	}
	if _, ok := hostMachOCPU("forge-unknown"); ok {
		t.Fatal("Mach-O mapping accepted an unknown architecture")
	}
}
