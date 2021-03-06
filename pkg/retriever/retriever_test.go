package retriever

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/vault"
	"github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

type mockSSMClient struct {
	Encoded bool
}

func (m *mockSSMClient) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {

	var v string

	if m.Encoded {
		v = base64.StdEncoding.EncodeToString([]byte("This is a base64 encoded CI test"))
	} else {
		v = "This is a CI test"
	}

	r := ssm.GetParameterOutput{
		Parameter: &ssmtypes.Parameter{
			ARN:              aws.String(""),
			DataType:         aws.String("text"),
			Name:             aws.String("test"),
			LastModifiedDate: aws.Time(time.Now()),
			Type:             ssmtypes.ParameterTypeString,
			Value:            &v,
		},
	}

	return &r, nil
}

func TestRetrievePlaintextSSMParameter(t *testing.T) {
	c := mockSSMClient{
		Encoded: false,
	}

	v, e := GetParameterFromSSM(context.TODO(), &c, logrus.New(), "/ci/test", false, false)

	assert.Equal(t, e, nil)
	assert.Equal(t, v, "This is a CI test")
}

func TestRetrieveEncodedSSMParameter(t *testing.T) {
	c := mockSSMClient{
		Encoded: true,
	}

	v, e := GetParameterFromSSM(context.TODO(), &c, logrus.New(), "/ci/example", false, true)

	assert.Equal(t, e, nil)
	assert.Equal(t, v, "This is a base64 encoded CI test")
}

func createVaultServer(t *testing.T) (net.Listener, *api.Client) {
	core, keyShares, rootToken := vault.TestCoreUnsealed(t)

	_ = keyShares

	ln, addr := http.TestServer(t, core)
	fmt.Printf("VAULT_ADDR=http://%s\n", ln.Addr().String())
	fmt.Printf("VAULT_TOKEN=%s\n", rootToken)

	conf := api.DefaultConfig()
	conf.Address = addr

	client, err := api.NewClient(conf)

	if err != nil {
		t.Fatal(err)
	}
	client.SetToken(rootToken)

	kvMount := &api.MountInput{
		Type:        "kv",
		Description: "CI Test",
		Options: map[string]string{
			"version": "2",
		},
	}

	s := client.Sys()
	s.Mount("kv-v2", kvMount)

	_, err = client.Logical().Write("kv-v2/data/test/ci/secret",
		map[string]interface{}{"data": map[string]string{"hello": "world", "config": "{\"some_key\": \"some_value\"}"}},
	)

	if err != nil {
		t.Fatal(err)
	}

	e1 := base64.StdEncoding.EncodeToString([]byte("{\"key_one\": \"value_one\"}"))

	e2 := base64.StdEncoding.EncodeToString([]byte("{\"key_two\": \"value_two\"}"))

	_, err = client.Logical().Write("kv-v2/data/test/ci/secret-encoded",
		map[string]interface{}{"data": map[string]string{"encoded_one": e1, "encoded_two": e2}},
	)

	if err != nil {
		t.Fatal(err)
	}

	return ln, client

}

func TestGetSecretFromVault(t *testing.T) {

	_, client := createVaultServer(t)

	log := logrus.New()

	v := client.Logical()

	r := GetSecretFromVault("kv-v2/data/test/ci/secret", false, log, v)
	if r["hello"] != "world" {
		t.Fatalf("Expected r['hello'] to be 'world' but received '%s'", r["hello"])
	}

	if r["config"] != "{\"some_key\": \"some_value\"}" {
		t.Fatalf("Expected r['config'] to be '{\"some_key\": \"some_value\"}' but received '%s'", r["config"])
	}
}

func TestGetEncodedSecretFromVault(t *testing.T) {

	_, client := createVaultServer(t)

	log := logrus.New()

	v := client.Logical()

	r := GetSecretFromVault("kv-v2/data/test/ci/secret-encoded", true, log, v)
	if r["encoded_one"] != "{\"key_one\": \"value_one\"}" {
		t.Fatalf("Expected r['encoded_one'] to be '{\"key_one\": \"value_one\"}' but received '%s'", r["encoded_one"])
	}

}
