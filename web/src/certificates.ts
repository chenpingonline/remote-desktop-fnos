export interface CertificateInfo {
  fingerprint:string; subject:string; issuer:string; notBefore:string; notAfter:string;
  verified:boolean; verification:string; canTrust:boolean;
}
export interface CertificateTrust {
  certificate:CertificateInfo;host:string;port:number;trustedAt:string;version:number;
}
export interface CertificateInspection {
  status:'valid'|'trusted'|'untrusted'|'changed'|'invalid';certificate:CertificateInfo;
  previous:CertificateTrust|null;check:string;expiresAt:string;
}
