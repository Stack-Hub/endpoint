/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package version

import (
	"testing"
)

func TestFullVersion(t *testing.T) {
	version := FullVersion()

	expected := Version + Build + " (" + GitCommit + ")"

	if version != expected {
		t.Fatalf("invalid version returned: %s", version)
	}
}