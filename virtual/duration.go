package virtual

import "time"

type Duration time.Duration

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d Duration) MarshalText() (text []byte, err error) {
	return []byte(time.Duration(d).String()), nil
}

func (d *Duration) UnmarshalText(text []byte) error {
	p, err := time.ParseDuration(string(text))
	*d = Duration(p)
	return err
}
