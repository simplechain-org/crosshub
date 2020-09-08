// Copyright 2016 The go-simplechain Authors
// This file is part of the go-simplechain library.
//
// The go-simplechain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-simplechain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-simplechain library. If not, see <http://www.gnu.org/licenses/>.

package core

//import (
//	"errors"
//	"github.com/hyperledger/fabric/bccsp"
//	"github.com/hyperledger/fabric/bccsp/idemix/bridge"
//	"github.com/hyperledger/fabric/bccsp/idemix/handlers"
//	"github.com/hyperledger/fabric/bccsp/sw"
//	"reflect"
//)
//
//// sigCache is used to cache the derived sender and contains
//// the signer used to derive it.
//type frSigCache struct {
//	*sw.CSP
//}
//
//func NewFrs(keyStore bccsp.KeyStore) (*frSigCache, error) {
//	base, err := sw.New(keyStore)
//	if err != nil {
//		return nil, err
//	}
//
//	csp := &frSigCache{CSP: base}
//
//	// key generators
//	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixIssuerKeyGenOpts{}), &handlers.IssuerKeyGen{Issuer: &bridge.Issuer{NewRand: bridge.NewRandOrPanic}})
//	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixUserSecretKeyGenOpts{}), &handlers.UserKeyGen{User: &bridge.User{NewRand: bridge.NewRandOrPanic}})
//	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixRevocationKeyGenOpts{}), &handlers.RevocationKeyGen{Revocation: &bridge.Revocation{}})
//
//	// key derivers
//	base.AddWrapper(reflect.TypeOf(handlers.NewUserSecretKey(nil, false)), &handlers.NymKeyDerivation{
//		User: &bridge.User{NewRand: bridge.NewRandOrPanic},
//	})
//
//	// signers
//	base.AddWrapper(reflect.TypeOf(handlers.NewUserSecretKey(nil, false)), &userSecreKeySignerMultiplexer{
//		signer:                  &handlers.Signer{SignatureScheme: &bridge.SignatureScheme{NewRand: bridge.NewRandOrPanic}},
//		nymSigner:               &handlers.NymSigner{NymSignatureScheme: &bridge.NymSignatureScheme{NewRand: bridge.NewRandOrPanic}},
//		credentialRequestSigner: &handlers.CredentialRequestSigner{CredRequest: &bridge.CredRequest{NewRand: bridge.NewRandOrPanic}},
//	})
//	base.AddWrapper(reflect.TypeOf(handlers.NewIssuerSecretKey(nil, false)), &handlers.CredentialSigner{
//		Credential: &bridge.Credential{NewRand: bridge.NewRandOrPanic},
//	})
//	base.AddWrapper(reflect.TypeOf(handlers.NewRevocationSecretKey(nil, false)), &handlers.CriSigner{
//		Revocation: &bridge.Revocation{},
//	})
//
//	// verifiers
//	base.AddWrapper(reflect.TypeOf(handlers.NewIssuerPublicKey(nil)), &issuerPublicKeyVerifierMultiplexer{
//		verifier:                  &handlers.Verifier{SignatureScheme: &bridge.SignatureScheme{NewRand: bridge.NewRandOrPanic}},
//		credentialRequestVerifier: &handlers.CredentialRequestVerifier{CredRequest: &bridge.CredRequest{NewRand: bridge.NewRandOrPanic}},
//	})
//	base.AddWrapper(reflect.TypeOf(handlers.NewNymPublicKey(nil)), &handlers.NymVerifier{
//		NymSignatureScheme: &bridge.NymSignatureScheme{NewRand: bridge.NewRandOrPanic},
//	})
//	base.AddWrapper(reflect.TypeOf(handlers.NewUserSecretKey(nil, false)), &handlers.CredentialVerifier{
//		Credential: &bridge.Credential{NewRand: bridge.NewRandOrPanic},
//	})
//	base.AddWrapper(reflect.TypeOf(handlers.NewRevocationPublicKey(nil)), &handlers.CriVerifier{
//		Revocation: &bridge.Revocation{},
//	})
//
//	// importers
//	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixUserSecretKeyImportOpts{}), &handlers.UserKeyImporter{
//		User: &bridge.User{},
//	})
//	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixIssuerPublicKeyImportOpts{}), &handlers.IssuerPublicKeyImporter{
//		Issuer: &bridge.Issuer{},
//	})
//	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixNymPublicKeyImportOpts{}), &handlers.NymPublicKeyImporter{
//		User: &bridge.User{},
//	})
//	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixRevocationPublicKeyImportOpts{}), &handlers.RevocationPublicKeyImporter{})
//
//	return csp, nil
//}
//
//func (csp *frSigCache) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error) {
//	// Validate arguments
//	if k == nil {
//		return nil, errors.New("Invalid Key. It must not be nil.")
//	}
//	// Do not check for digest
//
//	keyType := reflect.TypeOf(k)
//	signer, found := csp.Signers[keyType]
//	if !found {
//		return nil, errors.New("Unsupported 'SignKey' provided")
//	}
//
//	signature, err = signer.Sign(k, digest, opts)
//	if err != nil {
//		return nil, err
//	}
//
//	return
//}
//
//// Verify verifies signature against key k and digest
//// Notice that this is overriding the Sign methods of the sw impl. to avoid the digest check.
//func (csp *frSigCache) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (valid bool, err error) {
//	// Validate arguments
//	if k == nil {
//		return false, errors.New("Invalid Key. It must not be nil.")
//	}
//	if len(signature) == 0 {
//		return false, errors.New("Invalid signature. Cannot be empty.")
//	}
//	// Do not check for digest
//
//	verifier, found := csp.Verifiers[reflect.TypeOf(k)]
//	if !found {
//		return false, errors.New("Unsupported 'VerifyKey' provided")
//	}
//
//	valid, err = verifier.Verify(k, signature, digest, opts)
//	if err != nil {
//		return false, err
//	}
//
//	return
//}
//
//func SignFr(tx *FabricCross, s frSigCache, k bccsp.Key) (*FabricCross, error) {
//	sig, err := s.Sign(k,tx.Hash().Bytes(),nil)
//	if err != nil {
//		return nil, err
//	}
//	return tx.WithSignature(sig)
//}
//
//func FrSender(tx *FabricCross, s frSigCache,k bccsp.Key) (bool, error) {
//	return s.Verify(k,tx.Data.Proof,tx.Hash().Bytes(),nil)
//}

