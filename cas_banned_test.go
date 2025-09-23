package main

import "testing"

// User chip
func TestUserIsNotBanned(t *testing.T) {
	if checkBanned(1051416075) {
		t.Error("User should not banned")
	}
}

// https://cas.chat/query?u=1089155882
// TODO: This test is flaky because the user's ban status can change.
// It should be rewritten with a mock API call.
// func TestUserIsCasBanned(t *testing.T) {
// 	if !checkBanned(1089155882) {
// 		t.Error("User should banned")
// 	}
// }
