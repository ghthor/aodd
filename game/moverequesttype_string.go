// Code generated by "stringer -type=MoveRequestType"; DO NOT EDIT.

package game

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[MR_ERROR-0]
	_ = x[MR_MOVE-1]
	_ = x[MR_MOVE_CANCEL-2]
	_ = x[MR_SIZE-3]
}

const _MoveRequestType_name = "MR_ERRORMR_MOVEMR_MOVE_CANCELMR_SIZE"

var _MoveRequestType_index = [...]uint8{0, 8, 15, 29, 36}

func (i MoveRequestType) String() string {
	if i < 0 || i >= MoveRequestType(len(_MoveRequestType_index)-1) {
		return "MoveRequestType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _MoveRequestType_name[_MoveRequestType_index[i]:_MoveRequestType_index[i+1]]
}
