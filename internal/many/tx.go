package many

import (
	"fmt"
	"log/slog"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Arguments struct {
	From   string   `json:"from"`
	To     string   `json:"to"`
	Amount string   `json:"amount"`
	Symbol string   `json:"symbol"`
	Memo   []string `json:"memo"`
}

type TxInfo struct {
	Arguments Arguments `json:"argument"`
}

func GetTxInfo(r *resty.Client, hash string) (*TxInfo, error) {
	req := r.R().SetPathParam("thash", hash).SetResult(&TxInfo{})
	resp, err := req.Get("neighborhoods/{neighborhood}/transactions/{thash}")
	if err != nil {
		return nil, errors.WithMessage(err, "error getting MANY tx info")
	}

	txInfo := resp.Result().(*TxInfo)
	if txInfo == nil || (txInfo != nil && txInfo.Arguments.From == "") {
		return nil, fmt.Errorf("error unmarshalling MANY tx info")
	}
	slog.Debug("MANY tx info", "txInfo", txInfo)
	return txInfo, nil
}

func CheckTxInfo(txInfo *TxInfo, itemUUID uuid.UUID, manifestAddr string) error {
	// Check the MANY transaction `To` address
	if txInfo.Arguments.To != IllegalAddr {
		return fmt.Errorf("invalid MANY tx `to` address: %s", txInfo.Arguments.To)
	}

	// Check the MANY transaction `Memo`
	if len(txInfo.Arguments.Memo) != 2 {
		return fmt.Errorf("invalid MANY Memo length: %d", len(txInfo.Arguments.Memo))
	}

	// Check the MANY transaction UUID
	txUUID, err := uuid.Parse(txInfo.Arguments.Memo[0])
	if err != nil {
		return errors.WithMessagef(err, "invalid MANY tx UUID: %s", txInfo.Arguments.Memo[0])
	}

	// Check the Manifest destination address
	if txInfo.Arguments.Memo[1] != manifestAddr {
		return fmt.Errorf("invalid manifest destination address: %s", txInfo.Arguments.Memo[1])
	}

	// Check the MANY transaction UUID matches the work item UUID
	if txUUID != itemUUID {
		return fmt.Errorf("MANY tx UUID does not match work item UUID: %s, %s", txUUID, itemUUID)
	}

	return nil
}
