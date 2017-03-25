package user

import (
    "testing"
)

type testpair struct {
    userprefix string
    password string
}

var data = []testpair {
                        {"TrafficRouter", "1234567890"},
                        {"TrafficRouter", "0987654321"},
                      }


func TestAddSingleUserPasswd(t *testing.T) {

    u := NewUserWithPassword(data[0].userprefix, data[0].password)

    err := chkUser(u.Name)
    if err != nil {
        t.Error(
            "For", u.Name,
            "expected", "User to Exist",
            "got", err,
        )        
    }
    
    u.Delete()

}

func TestAddMultipleleUserPasswd(t *testing.T) {

}

func TestAddSingleUserKey(t *testing.T) {
}

func TestAddMultipleleUserKey(t *testing.T) {
}
