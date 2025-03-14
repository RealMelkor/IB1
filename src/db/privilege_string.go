// Code generated by "stringer -type Privilege"; DO NOT EDIT.

package db

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[NONE-0]
	_ = x[CREATE_BOARD-1]
	_ = x[ADMINISTRATION-2]
	_ = x[MANAGE_USER-3]
	_ = x[BAN_USER-4]
	_ = x[APPROVE_MEDIA-5]
	_ = x[BAN_MEDIA-6]
	_ = x[REMOVE_MEDIA-7]
	_ = x[REMOVE_POST-8]
	_ = x[HIDE_POST-9]
	_ = x[BYPASS_CAPTCHA-10]
	_ = x[BYPASS_MEDIA_APPROVAL-11]
	_ = x[VIEW_HIDDEN-12]
	_ = x[VIEW_PENDING_MEDIA-13]
	_ = x[VIEW_IP-14]
	_ = x[BAN_IP-15]
	_ = x[SHOW_RANK-16]
	_ = x[BYPASS_READONLY-17]
	_ = x[VIEW_PRIVATE-18]
	_ = x[USE_PRIVATE-19]
	_ = x[CREATE_POST-20]
	_ = x[CREATE_THREAD-21]
	_ = x[PIN_THREAD-22]
	_ = x[LAST-23]
}

const _Privilege_name = "NONECREATE_BOARDADMINISTRATIONMANAGE_USERBAN_USERAPPROVE_MEDIABAN_MEDIAREMOVE_MEDIAREMOVE_POSTHIDE_POSTBYPASS_CAPTCHABYPASS_MEDIA_APPROVALVIEW_HIDDENVIEW_PENDING_MEDIAVIEW_IPBAN_IPSHOW_RANKBYPASS_READONLYVIEW_PRIVATEUSE_PRIVATECREATE_POSTCREATE_THREADPIN_THREADLAST"

var _Privilege_index = [...]uint16{0, 4, 16, 30, 41, 49, 62, 71, 83, 94, 103, 117, 138, 149, 167, 174, 180, 189, 204, 216, 227, 238, 251, 261, 265}

func (i Privilege) String() string {
	if i < 0 || i >= Privilege(len(_Privilege_index)-1) {
		return "Privilege(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Privilege_name[_Privilege_index[i]:_Privilege_index[i+1]]
}
