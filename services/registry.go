package services

import (
	"golang.org/x/sys/windows/registry"
)

const userRegKey = `Software\AmneziaWG`

var userKey registry.Key

func openUserKey() (registry.Key, error) {
	if userKey != 0 {
		return userKey, nil
	}
	var err error
	userKey, err = registry.OpenKey(registry.CURRENT_USER, userRegKey, registry.QUERY_VALUE)
	if err != nil {
		return 0, err
	}
	return userKey, nil
}

func createUserKey() (registry.Key, error) {
	closeUserKey()
	var err error
	userKey, _, err = registry.CreateKey(registry.CURRENT_USER, userRegKey, registry.SET_VALUE)
	if err != nil {
		return 0, err
	}
	return userKey, nil
}

func closeUserKey() {
	if userKey == 0 {
		return
	}

	userKey.Close()
	userKey = 0
}

func UserKeyString(name string) (string, bool) {
	key, err := openUserKey()
	if err != nil {
		return err.Error(), false
	}
	val, _, err := key.GetStringValue(name)
	if err != nil {
		return err.Error(), false
	}
	return val, true
}

func SetUserKeyString(name string, value string) error {
	key, err := createUserKey()
	if err != nil {
		return err
	}
	defer closeUserKey()
	err = key.SetStringValue(name, value)
	if err != nil {
		return err
	}
	return nil
}
