package user

import (
    "testing"
    "io/ioutil"
)

const (
    PUBKEY = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDk7Ti7Fb1HmzvtWlDtKclvF9vChVdjp/fkdWdZR26HW54aNIh7YLwC1W8aNd6SUd2PEbUAjH6KujHVxA/dxsuYjQCaNouE+W3D+98UgJrfvG6O444BzUOplcHIUppp06f+utveH1gd3w8eyOQzSmLPTMkhKXvJTRuFgdytnmOh2A2qzE81v7I/ExPiIgdS6uBttFVFUvxBfjUpR6k8KnrmCYscJt4wzBQDPkKeI18K2ZNk8ig5389qlfGW/qRT+bxx0GE2UIaFfDIUL8zKp+KugZs0k1g3vCK/F6OKCWczigjnIoWCEK5txahfyMVv7rBTeK3vIq8X5gL2Lt0PWm1V root@digitalocean`
    PRIVKEY = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA5O04uxW9R5s77VpQ7SnJbxfbwoVXY6f35HVnWUduh1ueGjSI
e2C8AtVvGjXeklHdjxG1AIx+irox1cQP3cbLmI0AmjaLhPltw/vfFICa37xujuOO
Ac1DqZXByFKaadOn/rrb3h9YHd8PHsjkM0piz0zJISl7yU0bhYHcrZ5jodgNqsxP
Nb+yPxMT4iIHUurgbbRVRVL8QX41KUepPCp65gmLHCbeMMwUAz5CniNfCtmTZPIo
Od/PapXxlv6kU/m8cdBhNlCGhXwyFC/MyqfiroGbNJNYN7wivxejiglnM4oI5yKF
ghCubcWoX8jFb+6wU3it7yKvF+YC9i7dD1ptVQIDAQABAoIBAAHEex2mq1F0N2A7
xEgwfGepLX8w/F2+nEFdTrD3xjixUmUtZqhdNNZow3TWWaOTkjxajKU2cHutuFjI
LL8vm77Px+No7GbYbiqHNU+5NnjnwYrE4wHMjesvRtG/IYYTpkZnNu9eGpYQdNNu
BaUHu/+RvjPNWDFTsRS0zflhMa+8MZBl6Juubkim99NIBBx+dZrz9inTIYePlUMF
oiw8JJCTvlju9SW8oLkkOuRs9j89BP5VklUZCihK+rKDVedKD0sV76nTRYfORZae
MH4hmwwzM96XfYCETZc8cdnpEUTto62yDKAcWmGsp3zDi0UlEw7jAVmuwRQxDRbY
Lr/GHXkCgYEA/671W/r4q5opZVIl2Hy06BBlnhYtgympG2v5zQAB6HU4WRY68Em8
QS8/zD3ptUKdv/dpZ/CnocCI53v9mFMbUv2KNo6vgJnLHQYLpOHfItdIPXLVF6Sh
64HkB+nzaxW/IsPhATU5DgVlCurBBdJlUA8mAfB2kwhR/8F14QzMxG8CgYEA5TXI
RmjMGlVaxLJ379fyIB/NpRQu6Awjm/aG7oPXOAVnfpTx9aVnn1cNn4fC73/AQw8l
QHQI/3Tb+CYeLhcl9YXdkPkdzrw8A2dMEXx3NE18HRoZRaFd9LCJENI2x4B+s+Gf
dQqTlUNtLnJ8GE9zAzIT0RXkY2yacpAofo/GtHsCgYA3O0cbSHqhLxsUHQu52S6H
Fsusu6O3Oq+iEdATXZYL7g5vCCNRNsxo1FkWuKUcl7hV+I8Xed/sTBgG0Tz1w7Ya
VlSd9nKo+A/tRBoN0xENiK29QGoRwmmL4zIsF3iSwE7apq+bQDED+1xZYF6z8EAc
bDlMn/ItTtXPxq29ILO3FwKBgQCE3Bvu1Cgay4cFpP1ohR/QBx9IpN5bm024xbmI
39sMmfVXpjZqUSozbl5zLlqMQNzNAiZxqdDdYntu54lu5fQW0TWRJxVkFDAlOOca
666dHpzmsY4ckmDHyNxqZ69hDNZkpk+rpCnPx3muBqZv4P2lyI08ERiFmRoddfpD
AkwHqQKBgB+lglg9YVTavNepvRh5knKTEneVGappbybcN6z1AWQq6Pg53Ouq0WnN
dmwbNzHAFhZtstsdCqWfgu3TRzbhwIUSTtpa/jEgk4ZPO+gRv9nk00nRPvTR17pM
SEU+nRcQdT52CZC1kp3lSvhCKDmXo6+UWWmWy67jebN3QnfVZDn6
-----END RSA PRIVATE KEY-----`
)

type testpair struct {
    userprefix string
    password string
}

var data = []testpair {
                        {"tr", "1234567890"},
                        {"tr", "0987654321"},
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

    users := make(map[string]*User)
    
    for _, auth := range data {
        u := NewUserWithPassword(auth.userprefix, auth.password)
        users[u.Name] = u

        err := chkUser(u.Name)
        if err != nil {
            t.Error(
                "For", u.Name,
                "expected", "User to Exist",
                "got", err,
            )        
        }
    }
    
    for _, u := range users {
        u.Delete()                
    }
    
}

func TestAddSingleUserKey(t *testing.T) {
    
    err := ioutil.WriteFile("/tmp/user.pub", []byte(PUBKEY), 0644)
    if err != nil {
        t.Error("Error writing key file", err)  
    }
    
    u := NewUserWithKey(data[0].userprefix, "/tmp/user.pub")

    err = chkUser(u.Name)
    if err != nil {
        t.Error(
            "For", u.Name,
            "expected", "User to Exist",
            "got", err,
        )        
    }
    
    u.Delete()

}

func TestAddMultipleleUserKey(t *testing.T) {
    users := make(map[string]*User)

    err := ioutil.WriteFile("/tmp/user.pub", []byte(PUBKEY), 0644)
    if err != nil {
        t.Error("Error writing key file", err)  
    }
    
    for _, auth := range data {
        u := NewUserWithKey(auth.userprefix, "/tmp/user.pub")
        users[u.Name] = u

        err = chkUser(u.Name)
        if err != nil {
            t.Error(
                "For", u.Name,
                "expected", "User to Exist",
                "got", err,
            )        
        }
    }

    for _, u := range users {
        u.Delete()                
    }

}
