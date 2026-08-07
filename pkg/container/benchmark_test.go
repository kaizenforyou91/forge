package container

import "testing"

type BenchConfig struct {
	Name string
}

func BenchmarkResolve(b *testing.B) {

	c := New()

	_ = c.Register(&BenchConfig{
		Name: "Forge",
	})

	var cfg *BenchConfig

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Resolve(&cfg)
	}
}
func BenchmarkMakeGeneric(b *testing.B) {

	c := New()

	_ = c.Register(&BenchConfig{
		Name: "Forge",
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		_, _ = Make[*BenchConfig](c)

	}
}
func BenchmarkConstructor(b *testing.B) {

	c := New()

	_ = c.RegisterFactory(func() any {
		return &BenchConfig{
			Name: "Forge",
		}
	})

	var cfg *BenchConfig

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Resolve(&cfg)
	}
}
