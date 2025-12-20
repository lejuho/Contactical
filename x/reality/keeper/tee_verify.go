package keeper

import (
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/base64"
    "fmt"

    "github.com/mbreban/attestation"
)

func (k Keeper) ParseAndVerifyTEE(certBase64 string) (AttestationResult, error) {
    result := AttestationResult{}

    certBytes, err := base64.StdEncoding.DecodeString(certBase64)
    if err != nil {
        return result, fmt.Errorf("base64 디코딩 실패: %v", err)
    }

    cert, err := x509.ParseCertificate(certBytes)
    if err != nil {
        return result, fmt.Errorf("인증서 파싱 실패: %v", err)
    }

    var ext *pkix.Extension
    for i := range cert.Extensions {
        e := cert.Extensions[i]
        if e.Id.Equal(attestation.OIDKeyAttestationExtension) { // ✅ 올바른 OID 상수
            ext = &e
            break
        }
    }
    if ext == nil {
        return result, fmt.Errorf("안드로이드 attestation 확장을 찾지 못함")
    }

    // ✅ pkix.Extension 전체가 아니라 Value(der bytes)를 넘긴다
    attr, err := attestation.ParseExtension(ext.Value)
    if err != nil {
        return result, fmt.Errorf("안드로이드 증명 데이터 파싱 실패: %v", err)
    }

    // ✅ SecurityLevel 상수 이름
    result.IsHardwareBacked = attr.AttestationSecurityLevel >= attestation.TrustedEnvironment
    result.IsStrongBox      = attr.AttestationSecurityLevel == attestation.StrongBox

    tee := attr.TeeEnforced

    if tee.OsVersion != nil {
        result.OSVersion = *tee.OsVersion
    }
    if tee.OsPatchLevel != nil {
        result.PatchLevel = *tee.OsPatchLevel
    }

    if tee.RootOfTrust != nil {
        switch tee.RootOfTrust.VerifiedBootState {
        case attestation.Verified:
            result.VerifiedBoot = "Verified"
        case attestation.SelfSigned:
            result.VerifiedBoot = "SelfSigned"
        case attestation.Unverified:
            result.VerifiedBoot = "Unverified"
        case attestation.Failed:
            result.VerifiedBoot = "Failed"
        default:
            result.VerifiedBoot = "Unknown"
        }
    }

    return result, nil
}
