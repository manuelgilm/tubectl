package youtube

import (
	"os"
	"fmt"
	"encoding/json"
)

func SaveToken(path string, token *Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("Marshal token %w ", err)
	}

	err = os.WriteFile(path, data, 0666)
	if err != nil {
		return fmt.Errorf("saving token: %w ", err)
	}
	return nil
}

