package blockchain

type (
	TxInput struct {
		ID        []byte
		Out       int
		Signature string
	}

	TxOutput struct {
		Value  int
		PubKey string
	}
)

func (in *TxInput) CanUnlock(data string) bool       { return in.Signature == data }
func (out *TxOutput) CanBeUnlocked(data string) bool { return out.PubKey == data }
