package config

import "testing"

func BenchmarkLoad(b *testing.B) {

	cfg := Default()

	path := b.TempDir() + "/config.yaml"

	if err := Save(path, cfg); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		if _, err := Load(path); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkSave(b *testing.B) {

	cfg := Default()

	path := b.TempDir() + "/config.yaml"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		if err := Save(path, cfg); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkValidate(b *testing.B) {

	cfg := Default()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		if err := cfg.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkMerge(b *testing.B) {

	dst := Default()
	src := Default()

	src.Server.Port = 9090

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		tmp := dst

		Merge(&tmp, src)
	}
}
func BenchmarkEncryptConfig(b *testing.B) {

	cfg := Default()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		tmp := cfg

		if err := EncryptConfig(&tmp); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkDecryptConfig(b *testing.B) {

	cfg := Default()

	if err := EncryptConfig(&cfg); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		tmp := cfg

		if err := DecryptConfig(&tmp); err != nil {
			b.Fatal(err)
		}
	}
}
