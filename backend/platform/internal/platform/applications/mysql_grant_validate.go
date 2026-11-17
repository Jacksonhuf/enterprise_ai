package applications

import (
	"encoding/json"
	"errors"
)

func validateMySQLGrantPayload(g Grant) error {
	if !g.Active {
		return errors.New("grant insert requires active grant")
	}
	wrapped, err := json.Marshal([]Grant{g})
	if err != nil {
		return err
	}
	var object map[string]json.RawMessage
	var state State
	root := map[string]json.RawMessage{"grants": wrapped}
	raw, err := json.Marshal(root)
	if err != nil {
		return err
	}
	if json.Unmarshal(raw, &object) != nil || json.Unmarshal(raw, &state) != nil {
		return errors.New("grant payload invalid")
	}
	if _, err = planGrantSource(raw); err != nil {
		return err
	}
	return nil
}