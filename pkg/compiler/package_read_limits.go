package compiler

import "fmt"

const binaryMiB int64 = 1024 * 1024

// PackageReadLimits defines optional resource ceilings for ZIP package reads.
type PackageReadLimits struct {
	MaxArchiveBytes           int64
	MaxEntries                int
	MaxDocumentBytes          int64
	MaxArtifactBytes          int64
	MaxTotalUncompressedBytes int64
	RequireStoreCompression   bool
}

// Validate validates the bounded package-read configuration.
func (l PackageReadLimits) Validate() error {
	if l.MaxArchiveBytes <= 0 {
		return fmt.Errorf("%w: maximum archive bytes must be positive", ErrInvalidArtifactPackage)
	}
	if l.MaxEntries <= 0 {
		return fmt.Errorf("%w: maximum entries must be positive", ErrInvalidArtifactPackage)
	}
	if l.MaxDocumentBytes <= 0 {
		return fmt.Errorf("%w: maximum document bytes must be positive", ErrInvalidArtifactPackage)
	}
	if l.MaxArtifactBytes <= 0 {
		return fmt.Errorf("%w: maximum artifact bytes must be positive", ErrInvalidArtifactPackage)
	}
	if l.MaxTotalUncompressedBytes <= 0 {
		return fmt.Errorf("%w: maximum total uncompressed bytes must be positive", ErrInvalidArtifactPackage)
	}

	return nil
}

// AlphaPackageReadLimits returns the bounded package-read policy for Alpha
// package consumers. These are Alpha safety limits, not production guarantees.
func AlphaPackageReadLimits() PackageReadLimits {
	return PackageReadLimits{
		MaxArchiveBytes:           80 * binaryMiB,
		MaxEntries:                16,
		MaxDocumentBytes:          1 * binaryMiB,
		MaxArtifactBytes:          64 * binaryMiB,
		MaxTotalUncompressedBytes: 72 * binaryMiB,
		RequireStoreCompression:   true,
	}
}

// AlphaRuntimePackageReadLimits returns the bounded package-read policy for
// the Alpha runtime-loading path.
//
// It is retained for compatibility. New package consumers should use
// AlphaPackageReadLimits.
func AlphaRuntimePackageReadLimits() PackageReadLimits {
	return AlphaPackageReadLimits()
}
