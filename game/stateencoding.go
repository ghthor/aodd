package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ghthor/filu/rpg2d/entity"
	"github.com/ghthor/filu/rpg2d/quad/quadstate"
)

func init() {
	quadstate.StateEncode = stateEncodeJson
	quadstate.StateDecode = stateDecodeJson
}

type JsonTypePeeker struct {
	Type  string
	State map[string]interface{}
}

type ActorEntityStateJson struct {
	Type  string
	State ActorEntityState
}

type RemovedEntityStateJson struct {
	Type  string
	State entity.RemovedState
}

type StateDecodeJson struct {
	Type string
}

func stateEncodeJson(e entity.State, w io.Writer) error {
	var err error
	enc := json.NewEncoder(w)

	switch e := e.(type) {
	case WallEntityState:
		err = enc.Encode(e)
	case AssailEntityState:
		err = enc.Encode(e)
	case SayEntityState:
		err = enc.Encode(e)
	case ActorEntityState:
		err = enc.Encode(ActorEntityStateJson{
			Type:  "actor",
			State: e,
		})
	case entity.RemovedState:
		err = enc.Encode(RemovedEntityStateJson{
			Type:  "removed",
			State: e,
		})
	default:
		panic(fmt.Sprintf("unable to encode value %#v", e))
	}

	return err
}

func stateDecodeJson(data []byte) (entity.State, error) {
	var err error
	var peek JsonTypePeeker

	// TODO I think this is fixed and can be removed
	data = bytes.Trim(data, "\x00")
	err = json.Unmarshal(data, &peek)
	if err != nil {
		panic(fmt.Sprintf("error peeking type %s\n %#v\n%v", string(data), peek, err))
		return nil, err
	}

	switch peek.Type {
	case "wall":
		var result WallEntityState
		err = json.Unmarshal(data, &result)
		return result, err
	case "assail":
		var result AssailEntityState
		err = json.Unmarshal(data, &result)
		return result, err
	case "say":
		var result SayEntityState
		err = json.Unmarshal(data, &result)
		return result, err
	case "actor":
		var result ActorEntityStateJson
		err = json.Unmarshal(data, &result)
		return result.State, err
	case "removed":
		var result RemovedEntityStateJson
		err = json.Unmarshal(data, &result)
		return result.State, err
	default:
	}

	panic(fmt.Sprintf("decode peek Type unknown \"%s\"", peek.Type))
}
