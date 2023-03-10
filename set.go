package kargo

type Set struct {
	Name      string `yaml:"name"`
	Value     string `yaml:"value"`
	ValueFrom string `yaml:"valueFrom"`
}

func (s Set) AppendArgs(args []string, get GetValue) ([]string, error) {
	return s.appendArgs("--helm-set", args, get)
}

func (s Set) AppendArgoCDAppArgs(args []string, get GetValue) ([]string, error) {
	return s.appendArgs("--helm-set", args, get)
}

func (s Set) appendArgs(flag string, args []string, get GetValue) ([]string, error) {
	if s.ValueFrom != "" {
		v, err := get(s.ValueFrom)
		if err != nil {
			return nil, err
		}
		return append(args, flag, s.Name+"="+v), nil
	}

	return append(args, flag, s.Name+"="+s.Value), nil
}
