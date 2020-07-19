package di

type config struct {
	Singletone bool
}

type Option func(*config)

func Singletone() Option {
	return func(cfg *config) {
		cfg.Singletone = true
	}
}
