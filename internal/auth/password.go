package auth

import "golang.org/x/crypto/bcrypt"

// dummyHash 는 없는 이메일도 비밀번호 검사와 비슷한 시간이 걸리게 한다.
// 응답 속도만으로 가입된 메일인지 알아내지 못하게 하기 위한 장치다.
var dummyHash string

func init() {
	hashed, err := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	dummyHash = string(hashed)
}

func Hash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func Check(hash, password string) bool {
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
