/*
Copyright © 2021 Sniptt <support@sniptt.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/sniptt-official/snip/api/client"
	"github.com/sniptt-official/snip/build"
	"github.com/sniptt-official/snip/crypto/encrypt"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	defaultExpiry = 24 * time.Hour
	defaultRegion = "us-east-1"
)

var (
	expires time.Duration
	region  string

	otsCmd = &cobra.Command{
		Use:   "ots",
		Short: "Share encrypted secret via one-time URL",
		Long: `
Encrypts a secret and makes it available for sharing via one-time URL.

The secret is stored encrypted for a specified duration which can range
from 5 minutes to 7 days (default is 24 hours). The secret gets deleted
from the server upon retrieval therefore can only be viewed once.
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if expires.Minutes() < 5 {
				return errors.New("expiry must be at least 5 minutes")
			}

			if expires.Hours() > 168 {
				return errors.New("expiry must be less than 7 days")
			}

			if !isValidRegion(region) {
				return errors.New("invalid region")
			}

			bytes, err := getInputBytes()
			if err != nil {
				return err
			}

			encryptedBytes, err := encrypt.Bytes(bytes)
			if err != nil {
				return err
			}

			ciphertext, key := encryptedBytes.Ciphertext, encryptedBytes.Key

			ots, err := client.CreateOts(ciphertext, expires, region)
			if err != nil {
				return err
			}

			expiresAt := time.Unix(ots.ExpiresAt, 0)

			q := ots.ViewURL.Query()
			q.Set("ref", "ots-cli")
			q.Set("region", region)
			q.Set("v", build.Version)
			ots.ViewURL.RawQuery = q.Encode()
			ots.ViewURL.Fragment = base64.URLEncoding.EncodeToString(key)

			fmt.Printf(`
Your secret is now available on the below URL.

%v

You should only share this URL with the intended recipient.

Please note that once retrieved, the secret will no longer
be available for viewing. If not viewed, the secret will
automatically expire at approximately %v.
`,
				ots.ViewURL.String(),
				expiresAt.Format("2 Jan 2006 15:04:05"),
			)

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(otsCmd)

	otsCmd.Flags().DurationVarP(&expires, "expires", "x", defaultExpiry, "Secret will be deleted from the server after specified duration, supported units: s,m,h")
	otsCmd.Flags().StringVar(&region, "region", defaultRegion, "The region where secret should be created, supported regions: us-east-1,eu-central-1")
}

func getInputBytes() ([]byte, error) {
	fi, _ := os.Stdin.Stat() // Get the FileInfo struct describing the standard input.

	if (fi.Mode() & os.ModeCharDevice) == 0 {
		bytes, err := ioutil.ReadAll(os.Stdin)

		if err != nil {
			return nil, err
		}

		return bytes, nil
	} else {
		fmt.Print("Enter your secret: ")

		bytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return nil, err
		}

		return []byte(bytes), nil
	}
}

func isValidRegion(region string) bool {
	switch region {
	case
		"us-east-1",
		"eu-central-1":
		return true
	}
	return false
}
