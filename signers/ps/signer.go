/*
 * Copyright (c) SAS Institute Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ps

import (
	"errors"
	"io"
	"os"
	"strings"

	"gerrit-pdt.unx.sas.com/tools/relic.git/lib/authenticode"
	"gerrit-pdt.unx.sas.com/tools/relic.git/lib/certloader"
	"gerrit-pdt.unx.sas.com/tools/relic.git/lib/x509tools"
	"gerrit-pdt.unx.sas.com/tools/relic.git/signers"
	"gerrit-pdt.unx.sas.com/tools/relic.git/signers/pkcs"
)

var PsSigner = &signers.Signer{
	Name:      "ps",
	CertTypes: signers.CertTypeX509,
	TestPath:  testPath,
	Sign:      sign,
	Verify:    verify,
}

func init() {
	PsSigner.Flags().String("ps-style", "", "(Powershell) signature type")
	signers.Register(PsSigner)
}

func testPath(filepath string) bool {
	_, ok := authenticode.GetSigStyle(filepath)
	return ok
}

func sign(r io.Reader, cert *certloader.Certificate, opts signers.SignOpts) ([]byte, error) {
	argStyle, _ := opts.Flags.GetString("ps-style")
	if argStyle == "" {
		argStyle = opts.Path
	}
	style, err := getStyle(argStyle)
	if err != nil {
		return nil, err
	}
	digest, err := authenticode.DigestPowershell(r, style, opts.Hash)
	if err != nil {
		return nil, err
	}
	psd, err := digest.Sign(cert.Signer(), cert.Chain())
	if err != nil {
		return nil, err
	}
	blob, err := pkcs.Timestamp(psd, cert, opts, true)
	if err != nil {
		return nil, err
	}
	patch, err := digest.MakePatch(blob)
	if err != nil {
		return nil, err
	}
	return opts.SetBinPatch(patch)
}

func verify(f *os.File, opts signers.VerifyOpts) ([]*signers.Signature, error) {
	style, err := getStyle(f.Name())
	if err != nil {
		return nil, err
	}
	ts, err := authenticode.VerifyPowershell(f, style, opts.NoDigests)
	if err != nil {
		return nil, err
	}
	hash, _ := x509tools.PkixDigestToHash(ts.SignerInfo.DigestAlgorithm)
	return []*signers.Signature{&signers.Signature{
		Hash:          hash,
		X509Signature: ts,
	}}, nil
}

func getStyle(name string) (authenticode.PsSigStyle, error) {
	style, ok := authenticode.GetSigStyle(name)
	if !ok {
		return 0, errors.New("unknown powershell style, expected: " + strings.Join(authenticode.AllSigStyles(), " "))
	}
	return style, nil
}
