package taskflow

type options struct {
	name string
}

type Option interface {
	Apply(opt *options)
}

type nameOption string

func (n nameOption) Apply(opt *options) {
	opt.name = string(n)
}

func WithName(name string) Option {
	return nameOption(name)
}
