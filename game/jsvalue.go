// +build js,wasm

package game

import "syscall/js"

func (e ActorEntityState) JSValue() js.Value {
	v := js.Global().Get("Object").New()

	v.Set("Id", int64(e.Id))
	v.Set("Name", e.Name)

	// Movement and position
	v.Set("Facing", int(e.Facing))
	v.Set("Cell", e.Cell)

	if e.PathAction != nil {
		v.Set("PathAction", e.PathAction)
	} else {
		v.Set("PathAction", js.Null())
	}

	// Health and Mana
	v.Set("Hp", e.Hp)
	v.Set("HpMax", e.HpMax)
	v.Set("Mp", e.Mp)
	v.Set("MpMax", e.MpMax)

	return v
}

func (e WallEntityState) JSValue() js.Value {
	v := js.Global().Get("Object").New()
	v.Set("Type", e.Type)
	v.Set("Id", int64(e.Id))
	v.Set("Cell", e.Cell)
	return v
}

func (e AssailEntityState) JSValue() js.Value {
	v := js.Global().Get("Object").New()
	v.Set("Type", e.Type)

	v.Set("Id", int64(e.Id))

	v.Set("SpawnedBy", int64(e.SpawnedBy))
	v.Set("SpawnedAt", int64(e.SpawnedAt))

	v.Set("Cell", e.Cell)
	return v
}

func (e SayEntityState) JSValue() js.Value {
	v := js.Global().Get("Object").New()
	v.Set("Type", e.Type)

	v.Set("Id", int64(e.Id))

	v.Set("SaidBy", int64(e.SaidBy))
	v.Set("SaidAt", int64(e.SaidAt))

	v.Set("Cell", e.Cell)

	v.Set("Msg", e.Msg)
	return v
}
