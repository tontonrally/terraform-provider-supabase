// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/oapi-codegen/nullable"
	"github.com/supabase/cli/pkg/api"
	"github.com/supabase/terraform-provider-supabase/examples"
	"gopkg.in/h2non/gock.v1"
)

const testProjectRef = "mayuaycdtijbctgqbycg" //nolint:gosec

func TestAccApiKeyResource(t *testing.T) {
	// Setup mock api
	defer gock.OffAll()
	// Step 1: create
	testApiKeyUUID := uuid.New()
	apiKeysEndpoint := fmt.Sprintf("/v1/projects/%s/api-keys", testProjectRef)
	apiKeyEndpoint := fmt.Sprintf("%s/%s", apiKeysEndpoint, testApiKeyUUID.String())
	gock.New("https://api.supabase.com").
		Get(apiKeysEndpoint).
		Reply(http.StatusOK).
		JSON([]api.ApiKeyResponse{
			{
				Name:   "anon",
				Type:   nullable.NewNullableWithValue(api.ApiKeyResponseType("legacy")),
				ApiKey: nullable.NewNullableWithValue("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.anon"),
			},
			{
				Name:   "service_role",
				Type:   nullable.NewNullableWithValue(api.ApiKeyResponseType("legacy")),
				ApiKey: nullable.NewNullableWithValue("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.service_role"),
			},
		})
	gock.New("https://api.supabase.com").
		Post(apiKeysEndpoint).
		Reply(http.StatusCreated).
		JSON(api.ApiKeyResponse{
			Id:     nullable.NewNullableWithValue(uuid.New().String()),
			Name:   "default",
			Type:   nullable.NewNullableWithValue(api.ApiKeyResponseType("publishable")),
			ApiKey: nullable.NewNullableWithValue("sb_publishable_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"),
		})
	gock.New("https://api.supabase.com").
		Post(apiKeysEndpoint).
		Reply(http.StatusCreated).
		JSON(api.ApiKeyResponse{
			Id:     nullable.NewNullableWithValue(testApiKeyUUID.String()),
			Name:   "test",
			Type:   nullable.NewNullableWithValue(api.ApiKeyResponseType("secret")),
			ApiKey: nullable.NewNullableWithValue("sb_secret_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"),
		})
	gock.New("https://api.supabase.com").
		Get(apiKeyEndpoint).
		Persist().
		Reply(http.StatusOK).
		JSON(api.ApiKeyResponse{
			Id:     nullable.NewNullableWithValue(testApiKeyUUID.String()),
			Name:   "test",
			Type:   nullable.NewNullableWithValue(api.ApiKeyResponseType("secret")),
			ApiKey: nullable.NewNullableWithValue("sb_secret_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"),
			SecretJwtTemplate: nullable.NewNullableWithValue(map[string]interface{}{
				"role": "service_role",
			}),
		})
	gock.New("https://api.supabase.com").
		Delete(apiKeyEndpoint).
		Reply(http.StatusOK)

	// Run test
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: examples.ApiKeyResourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supabase_apikey.new", "id", testApiKeyUUID.String()),
				),
			},
			// ImportState testing
			{
				ResourceName:            "supabase_apikey.new",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"name", "project_ref"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["supabase_apikey.new"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					projectRef, ok := rs.Primary.Attributes["project_ref"]
					if !ok || projectRef == "" {
						return "", fmt.Errorf("project_ref not found in state")
					}
					if rs.Primary.ID == "" {
						return "", fmt.Errorf("id not set in state")
					}
					return fmt.Sprintf("%s,%s", projectRef, rs.Primary.ID), nil
				},
			},
			// Update and Read testing
			{
				Config: testAccApikeyResourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supabase_apikey.new", "name", "test"),
					resource.TestCheckResourceAttr("supabase_apikey.new", "project_ref", testProjectRef),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

const testAccApikeyResourceConfig = `
resource "supabase_apikey" "new" {
  project_ref = "` + testProjectRef + `"
  name        = "test"
}
`
