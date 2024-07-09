package main

import (
	"fmt"
	"os/user"
)

// Group holds the gid of the sysadmin and helpdesk groups and maps from the zammad ID to system IDs for science.
var Group = map[int]uint32{
	1: 40016,
	2: 2321,
}

// User holds the mapping of Zammad ID to system ids. Only works on maker.
var User = map[int]uint32{
	4:  11878, // john 4
	5:  0,     // polman 5
	6:  0,     // remcoa 6
	7:  0,     // bjorn 7
	8:  0,     // stefan 8
	9:  0,     // petervc 9
	10: 8451,  // bram 10
	11: 0,     // ericl 11
	12: 0,     // dominic 12
	13: 41090, // miek 13
	14: 0,     // visser 14
	15: 0,     // tcunnen 15
	16: 21365, // simon 16
	17: 0,     // wim 17
	18: 0,     // arnoudt 18
	44: 0,     // bbellink 44
}

// UID finds the zamad user using the *username*
func UID(uid uint32) int {
	u, err := user.LookupId(fmt.Sprintf("%d", uid))
	if err != nil {
		return 65534
	}
	switch u.Username {
	case "miek":
		return 13
	case "bram":
		return 10
	case "simon":
		return 16
	}
	return 65534
}
