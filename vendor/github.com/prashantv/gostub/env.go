package gostub

import "os"

func (s *Stubs) checkEnvKey(k string) {
	if _, ok := s.origEnv[k]; !ok {
		v, ok := os.LookupEnv(k)
		s.origEnv[k] = envVal{v, ok}
	}
}

// SetEnv the specified environent variable to the specified value.
func (s *Stubs) SetEnv(k, v string) *Stubs {
	s.checkEnvKey(k)

	os.Setenv(k, v)
	return s
}

// UnsetEnv unsets the specified environent variable.
func (s *Stubs) UnsetEnv(k string) *Stubs {
	s.checkEnvKey(k)

	os.Unsetenv(k)
	return s
}

func (s *Stubs) resetEnv() {
	for k, v := range s.origEnv {
		if v.ok {
			os.Setenv(k, v.val)
		} else {
			os.Unsetenv(k)
		}
	}
}
