// Code generated by "stringer -type=UseRequestType"; DO NOT EDIT.

package game

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[UR_ERROR-0]
	_ = x[UR_USE-1]
	_ = x[UR_USE_CANCEL-2]
	_ = x[UR_SIZE-3]
}

const _UseRequestType_name = "UR_ERRORUR_USEUR_USE_CANCELUR_SIZE"

var _UseRequestType_index = [...]uint8{0, 8, 14, 27, 34}

func (i UseRequestType) String() string {
	if i < 0 || i >= UseRequestType(len(_UseRequestType_index)-1) {
		return "UseRequestType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _UseRequestType_name[_UseRequestType_index[i]:_UseRequestType_index[i+1]]
}
