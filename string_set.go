package main

type StringSet struct {
	value []string
}

func (f *StringSet) String() string {
	return ""
}

func (f *StringSet) Set(s string) error {
	f.value = append(f.value, s)
	return nil
}
