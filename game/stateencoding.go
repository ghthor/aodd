package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/entity"
	"github.com/ghthor/filu/rpg2d/quadstate"
	"github.com/ghthor/filu/sim/stime"
)

func init() {
	quadstate.StateEncode = stateEncodeBinary
	quadstate.StateDecode = stateDecodeBinary
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

type EncodedStateType uint

const (
	TypeWallEntity EncodedStateType = iota
	TypeAssailEntity
	TypeSayEntity
	TypeActorEntity
	TypeRemovedEntity
)

type Cell coord.Cell
type PathAction coord.PathActionState

func (c Cell) BinaryWriteTo(w io.Writer) (err error) {
	err = binary.Write(w, binary.LittleEndian, int32(c.X))
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.LittleEndian, int32(c.Y))
	return
}

func MustCellReadFrom(r io.Reader) (result coord.Cell) {
	var X, Y int32
	must(binary.Read(r, binary.LittleEndian, &X))
	must(binary.Read(r, binary.LittleEndian, &Y))
	result.X = int(X)
	result.Y = int(Y)
	return
}

func (p PathAction) BinaryWriteTo(w io.Writer) (err error) {
	err = binary.Write(w, binary.LittleEndian, int64(p.Start))
	err = binary.Write(w, binary.LittleEndian, int64(p.End))
	err = Cell(p.Orig).BinaryWriteTo(w)
	err = Cell(p.Dest).BinaryWriteTo(w)
	return err
}

func MustPathActionReadFrom(r io.Reader) (result coord.PathActionState) {
	var start, end int64
	must(binary.Read(r, binary.LittleEndian, &start))
	must(binary.Read(r, binary.LittleEndian, &end))
	result.Start = stime.Time(start)
	result.End = stime.Time(end)
	result.Orig = MustCellReadFrom(r)
	result.Dest = MustCellReadFrom(r)
	return
}

func stateEncodeBinary(e entity.State, w io.Writer) error {
	var err error

	switch e := e.(type) {
	case WallEntityState:
		must(binary.Write(w, binary.LittleEndian, uint8(TypeWallEntity)))
		must(binary.Write(w, binary.LittleEndian, int64(e.Id)))
		must(Cell(e.Cell).BinaryWriteTo(w))
	case AssailEntityState:
		must(binary.Write(w, binary.LittleEndian, uint8(TypeAssailEntity)))
		must(binary.Write(w, binary.LittleEndian, int64(e.Id)))
		must(binary.Write(w, binary.LittleEndian, int64(e.SpawnedBy)))
		must(binary.Write(w, binary.LittleEndian, int64(e.SpawnedAt)))
		must(Cell(e.Cell).BinaryWriteTo(w))
	case SayEntityState:
		must(binary.Write(w, binary.LittleEndian, uint8(TypeSayEntity)))
		must(binary.Write(w, binary.LittleEndian, int64(e.Id)))
		must(binary.Write(w, binary.LittleEndian, int64(e.SaidBy)))
		must(binary.Write(w, binary.LittleEndian, int64(e.SaidAt)))
		must(Cell(e.Cell).BinaryWriteTo(w))
		_, err = io.WriteString(w, e.Msg)
	case ActorEntityState:
		must(binary.Write(w, binary.LittleEndian, uint8(TypeActorEntity)))
		must(binary.Write(w, binary.LittleEndian, int64(e.Id)))
		must(binary.Write(w, binary.LittleEndian, byte(e.Facing)))
		must(Cell(e.Cell).BinaryWriteTo(w))
		must(binary.Write(w, binary.LittleEndian, e.PathAction != nil))
		if e.PathAction != nil {
			must(PathAction(*e.PathAction).BinaryWriteTo(w))
		}
		must(binary.Write(w, binary.LittleEndian, int16(e.Hp)))
		must(binary.Write(w, binary.LittleEndian, int16(e.HpMax)))
		must(binary.Write(w, binary.LittleEndian, int16(e.Mp)))
		must(binary.Write(w, binary.LittleEndian, int16(e.MpMax)))
		_, err = io.WriteString(w, e.Name)

	case entity.RemovedState:
		must(binary.Write(w, binary.LittleEndian, uint8(TypeRemovedEntity)))
		must(binary.Write(w, binary.LittleEndian, int64(e.Id)))
		must(Cell(e.Cell).BinaryWriteTo(w))
	default:
		panic(fmt.Sprintf("unable to encode value %#v", e))
	}

	return err
}

func must(e error) {
	if e != nil {
		panic(e)
	}
}

func mustReadAll(r io.Reader) []byte {
	b, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return b
}

func stateDecodeBinary(data []byte) (entity.State, error) {
	var err error

	r := bytes.NewReader(data)

	var peek uint8
	err = binary.Read(r, binary.LittleEndian, &peek)

	var entityId int64
	var time int64

	switch EncodedStateType(peek) {
	case TypeWallEntity:
		var result WallEntityState
		result.Type = "wall"
		must(binary.Read(r, binary.LittleEndian, &entityId))
		result.Id = entity.Id(entityId)
		result.Cell = MustCellReadFrom(r)
		return result, err
	case TypeAssailEntity:
		var result AssailEntityState
		result.Type = "assail"
		must(binary.Read(r, binary.LittleEndian, &entityId))
		result.Id = entity.Id(entityId)
		must(binary.Read(r, binary.LittleEndian, &entityId))
		result.SpawnedBy = entity.Id(entityId)
		must(binary.Read(r, binary.LittleEndian, &time))
		result.SpawnedAt = stime.Time(time)
		result.Cell = MustCellReadFrom(r)
		return result, err
	case TypeSayEntity:
		var result SayEntityState
		result.Type = "say"
		must(binary.Read(r, binary.LittleEndian, &entityId))
		result.Id = entity.Id(entityId)
		must(binary.Read(r, binary.LittleEndian, &entityId))
		result.SaidBy = entity.Id(entityId)
		must(binary.Read(r, binary.LittleEndian, &time))
		result.SaidAt = stime.Time(time)
		result.Cell = MustCellReadFrom(r)
		result.Msg = string(mustReadAll(r))
		return result, err
	case TypeActorEntity:
		var result ActorEntityState
		must(binary.Read(r, binary.LittleEndian, &entityId))
		result.Id = entity.Id(entityId)
		var facing byte
		must(binary.Read(r, binary.LittleEndian, &facing))
		result.Facing = coord.Direction(facing)
		result.Cell = MustCellReadFrom(r)
		var pathNotNull bool
		must(binary.Read(r, binary.LittleEndian, &pathNotNull))
		if pathNotNull {
			pa := MustPathActionReadFrom(r)
			result.PathAction = &pa
		}
		var hp, hpMax, mp, mpMax int16
		must(binary.Read(r, binary.LittleEndian, &hp))
		must(binary.Read(r, binary.LittleEndian, &hpMax))
		must(binary.Read(r, binary.LittleEndian, &mp))
		must(binary.Read(r, binary.LittleEndian, &mpMax))
		result.Hp = int(hp)
		result.HpMax = int(hpMax)
		result.Mp = int(mp)
		result.MpMax = int(mpMax)
		result.Name = string(mustReadAll(r))
		return result, err
	case TypeRemovedEntity:
		var result entity.RemovedState
		must(binary.Read(r, binary.LittleEndian, &entityId))
		result.Id = entity.Id(entityId)
		result.Cell = MustCellReadFrom(r)
		return result, err
	default:
	}

	panic(fmt.Sprintf("decode peek Type unknown \"%v\"", peek))
}
