package vault

type Vault struct {
	LiveFuncs map[string]interface{}
	LiveTypes map[string]func() interface{}

	DataBag   map[string]interface{}
	Extension interface{}
}
