// generated by stringer -type=ResponseMsg; DO NOT EDIT

package game

import "fmt"

const _ResponseMsg_name = "RESP_AUTH_FAILEDRESP_ACTOR_DOESNT_EXISTRESP_LOGIN_SUCCESSRESP_CREATE_SUCCESS"

var _ResponseMsg_index = [...]uint8{0, 16, 39, 57, 76}

func (i ResponseMsg) String() string {
	if i < 0 || i+1 >= ResponseMsg(len(_ResponseMsg_index)) {
		return fmt.Sprintf("ResponseMsg(%d)", i)
	}
	return _ResponseMsg_name[_ResponseMsg_index[i]:_ResponseMsg_index[i+1]]
}
