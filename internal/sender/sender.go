package sender

type Sender struct {
	filepath string
}

func NewSender(filepath string) *Sender {
	return &Sender{
		filepath: filepath,
	}
}
