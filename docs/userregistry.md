# Registry Keys for Users

#### `HKCU\Software\AmneziaWG\Language`

When this key is set, the UI will try to use the specified language as display language.

```
> reg add HKCU\Software\AmneziaWG /v LimitedOperatorUI /t REG_SZ /d "en" /f
```
