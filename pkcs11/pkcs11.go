package main

/*
#include <stdio.h>
#include <stdlib.h>
#include "pkcs11.h"
*/
import (
	"C"
)
import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"golang.org/x/crypto/ssh"
	"log"
	"log/syslog"
	"sync"
	"unsafe"

	"github.com/agrinman/krssh"
)

//export C_GetFunctionList
func C_GetFunctionList(l **C.CK_FUNCTION_LIST) C.CK_RV {
	logwriter, e := syslog.New(syslog.LOG_NOTICE, "krssh-pkcs11")
	if e == nil {
		log.SetOutput(logwriter)
	}
	log.Println("getFunctionList")
	*l = &functions
	return C.CKR_OK
}

//export C_Initialize
func C_Initialize(C.CK_VOID_PTR) C.CK_RV {
	log.Println("initialize")
	return C.CKR_OK
}

//export C_GetInfo
func C_GetInfo(ck_info *C.CK_INFO) C.CK_RV {
	log.Println("getInfo")
	*ck_info = C.CK_INFO{
		cryptokiVersion: C.struct__CK_VERSION{
			major: 2,
			minor: 20,
		},
		flags:              0,
		manufacturerID:     bytesToChar32([]byte("KryptCo Inc.")),
		libraryDescription: bytesToChar32([]byte("kryptonite pkcs11 middleware")),
		libraryVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
	}
	return C.CKR_OK
}

//export C_GetSlotList
func C_GetSlotList(token_present C.uchar, slot_list *C.CK_SLOT_ID, count *C.ulong) C.CK_RV {
	log.Println("getSlotList input count", *count)
	if slot_list == nil {
		log.Println("slot_list nil")
		//	just return count
		*count = 1
		return C.CKR_OK
	}
	if *count == 0 {
		log.Println("buffer too small")
		return C.CKR_BUFFER_TOO_SMALL
	}
	*count = 1
	*slot_list = 0
	return C.CKR_OK
}

//export C_GetSlotInfo
func C_GetSlotInfo(slotID C.CK_SLOT_ID, slotInfo *C.CK_SLOT_INFO) C.CK_RV {
	log.Println("getSlotInfo")
	*slotInfo = C.CK_SLOT_INFO{
		manufacturerID:  bytesToChar32([]byte("KryptCo Inc.")),
		slotDescription: bytesToChar64([]byte("kryptonite pkcs11 middleware")),
		hardwareVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
		firmwareVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
		//	TODO: for now, always present
		flags: C.CKF_TOKEN_PRESENT | C.CKF_REMOVABLE_DEVICE,
	}

	return C.CKR_OK
}

//export C_GetTokenInfo
func C_GetTokenInfo(slotID C.CK_SLOT_ID, tokenInfo *C.CK_TOKEN_INFO) C.CK_RV {
	log.Println("getTokenInfo")
	*tokenInfo = C.CK_TOKEN_INFO{
		label:               bytesToChar32([]byte("kryptonite iOS")),
		manufacturerID:      bytesToChar32([]byte("KryptCo Inc.")),
		model:               bytesToChar16([]byte("kryptonite iOS")),
		serialNumber:        bytesToChar16([]byte("1")),
		ulMaxSessionCount:   16,
		ulSessionCount:      0,
		ulMaxRwSessionCount: 16,
		ulRwSessionCount:    0,
		ulMaxPinLen:         0,
		ulMinPinLen:         0,
		hardwareVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
		firmwareVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
		flags: C.CKF_PROTECTED_AUTHENTICATION_PATH | C.CKF_TOKEN_INITIALIZED,
	}
	return C.CKR_OK
}

//export C_OpenSession
func C_OpenSession(slotID C.CK_SLOT_ID, flags C.CK_FLAGS, pApplication C.CK_VOID_PTR,
	notify C.CK_NOTIFY, sessionHandle C.CK_SESSION_HANDLE_PTR) C.CK_RV {
	log.Println("openSession")
	if flags&C.CKF_SERIAL_SESSION == 0 {
		log.Println("CKF_SERIAL_SESSION not set")
		return C.CKR_SESSION_PARALLEL_NOT_SUPPORTED
	}
	if notify != nil {
		log.Println("notify callback passed")
	}
	return C.CKR_OK
}

//export C_GetSessionInfo
func C_GetSessionInfo(session C.CK_SESSION_HANDLE, info *C.CK_SESSION_INFO) C.CK_RV {
	log.Println("GetSessionInfo")
	*info = C.CK_SESSION_INFO{
		slotID: 0,
		state:  C.CKS_RW_USER_FUNCTIONS,
		flags:  C.CKF_RW_SESSION | C.CKF_SERIAL_SESSION,
	}
	return C.CKR_OK
}

var mechanismTypes []C.CK_MECHANISM_TYPE = []C.CK_MECHANISM_TYPE{
	C.CKM_RSA_PKCS,
	C.CKM_SHA256_RSA_PKCS,
}

//export C_GetMechanismList
func C_GetMechanismList(slotID C.CK_SLOT_ID, mechList *C.CK_MECHANISM_TYPE, count *C.CK_ULONG) C.CK_RV {
	if mechList == nil {
		*count = C.CK_ULONG(len(mechanismTypes))
		return C.CKR_OK
	}
	if *count < C.CK_ULONG(len(mechanismTypes)) {
		return C.CKR_BUFFER_TOO_SMALL
	}
	for i := C.CK_ULONG(0); i < *count; i++ {
		*mechList = mechanismTypes[i]
		mechList = (*C.CK_MECHANISM_TYPE)(unsafe.Pointer(uintptr(unsafe.Pointer(mechList)) + unsafe.Sizeof(*mechList)))
	}
	log.Println("C_GetMechanismList")
	return C.CKR_OK
}

//export C_GetMechanismInfo
func C_GetMechanismInfo(slotID C.CK_SLOT_ID, _type C.CK_MECHANISM_TYPE, info *C.CK_MECHANISM_INFO) C.CK_RV {
	log.Println("C_GetMechanismInfo")
	if _type == C.CKM_RSA_PKCS {
		log.Println("CKM_RSA_PKCS")
		*info = C.CK_MECHANISM_INFO{
			ulMinKeySize: 4096,
			ulMaxKeySize: 4096,
			flags:        C.CKF_SIGN | C.CKF_HW,
		}
	}
	return C.CKR_OK
}

//export C_CloseSession
func C_CloseSession(session C.CK_SESSION_HANDLE) C.CK_RV {
	log.Println("closeSession")
	mutex.Lock()
	defer mutex.Unlock()
	return C.CKR_OK
}

var sessionFindObjectTypes map[C.CK_SESSION_HANDLE][]C.CK_ATTRIBUTE = map[C.CK_SESSION_HANDLE][]C.CK_ATTRIBUTE{}
var mutex sync.Mutex

var sessionFoundObjects map[C.CK_SESSION_HANDLE]map[C.CK_OBJECT_HANDLE]bool = map[C.CK_SESSION_HANDLE]map[C.CK_OBJECT_HANDLE]bool{}
var sessionFindingObjects map[C.CK_SESSION_HANDLE]map[C.CK_OBJECT_HANDLE]bool = map[C.CK_SESSION_HANDLE]map[C.CK_OBJECT_HANDLE]bool{}

func findOnce(session C.CK_SESSION_HANDLE, object C.CK_OBJECT_HANDLE) {
	if _, ok := sessionFindingObjects[session]; !ok {
		sessionFindingObjects[session] = map[C.CK_OBJECT_HANDLE]bool{}
	}
	if _, ok := sessionFoundObjects[session]; !ok {
		sessionFoundObjects[session] = map[C.CK_OBJECT_HANDLE]bool{}
	}
	if found, ok := sessionFoundObjects[session][object]; ok && found {
		return
	}
	sessionFindingObjects[session][object] = true
}

func found(session C.CK_SESSION_HANDLE, object C.CK_OBJECT_HANDLE) {
	if _, ok := sessionFindingObjects[session]; !ok {
		sessionFindingObjects[session] = map[C.CK_OBJECT_HANDLE]bool{}
	}
	if _, ok := sessionFoundObjects[session]; !ok {
		sessionFoundObjects[session] = map[C.CK_OBJECT_HANDLE]bool{}
	}
	delete(sessionFindingObjects[session], object)
	sessionFoundObjects[session][object] = true
}

//export C_FindObjectsInit
func C_FindObjectsInit(session C.CK_SESSION_HANDLE, templates C.CK_ATTRIBUTE_PTR, count C.CK_ULONG) C.CK_RV {
	log.Println("FindObjectsInit")
	mutex.Lock()
	defer mutex.Unlock()
	if count == 0 {
		log.Println("count == 0")
		findOnce(session, PUBKEY_HANDLE)
		findOnce(session, PRIVKEY_HANDLE)
		return C.CKR_OK
	}
	for i := C.CK_ULONG(0); i < count; i++ {
		log.Println(templates._type)
		switch templates._type {
		case C.CKA_CLASS:
			switch *(*C.CK_OBJECT_CLASS)(templates.pValue) {
			case C.CKO_PUBLIC_KEY:
				log.Println("init search for CKO_PUBLIC_KEY")
				findOnce(session, PUBKEY_HANDLE)
			case C.CKO_PRIVATE_KEY:
				log.Println("init search for CKO_PRIVATE_KEY")
				findOnce(session, PRIVKEY_HANDLE)
			}
		case C.CKO_MECHANISM:
			log.Println("init search for CKO_MECHANISM")
		}
		templates = C.CK_ATTRIBUTE_PTR(unsafe.Pointer(uintptr(unsafe.Pointer(templates)) + unsafe.Sizeof(*templates)))
	}
	return C.CKR_OK
}

const PUBKEY_HANDLE C.CK_OBJECT_HANDLE = 1
const PRIVKEY_HANDLE C.CK_OBJECT_HANDLE = 2

var PUBKEY_ID []byte = []byte{1}

//export C_FindObjects
func C_FindObjects(session C.CK_SESSION_HANDLE, objects C.CK_OBJECT_HANDLE_PTR, maxCount C.CK_ULONG, count C.CK_ULONG_PTR) C.CK_RV {
	log.Println("FindObjects")
	//	TODO: error handle here
	mutex.Lock()
	defer mutex.Unlock()
	remainingCount := maxCount
	foundCount := C.CK_ULONG(0)
	for handle, _ := range sessionFindingObjects[session] {
		switch handle {
		case PUBKEY_HANDLE:
			*objects = PUBKEY_HANDLE
			found(session, PUBKEY_HANDLE)
		case PRIVKEY_HANDLE:
			*objects = PRIVKEY_HANDLE
			found(session, PRIVKEY_HANDLE)
		}
		foundCount++
		remainingCount--
		if remainingCount == 0 {
			break
		}
		objects = (*C.CK_OBJECT_HANDLE)(unsafe.Pointer((uintptr(unsafe.Pointer(objects)) + unsafe.Sizeof(*objects))))
	}
	*count = foundCount
	return C.CKR_OK
}

//export C_FindObjectsFinal
func C_FindObjectsFinal(session C.CK_SESSION_HANDLE) C.CK_RV {
	return C.CKR_OK
}

var staticMe = krssh.Profile{}

//export C_GetAttributeValue
func C_GetAttributeValue(session C.CK_SESSION_HANDLE, object C.CK_OBJECT_HANDLE, template C.CK_ATTRIBUTE_PTR, count C.CK_ULONG) C.CK_RV {
	mutex.Lock()
	defer mutex.Unlock()
	log.Println("C_GetAttributeValue")
	me, err := getMe()
	if err != nil {
		log.Println("getMe error " + err.Error())
		return C.CKR_GENERAL_ERROR
	}
	pk, err := me.RSAPublicKey()
	if err != nil {
		log.Println("me.RSAPublicKey error " + err.Error())
		return C.CKR_GENERAL_ERROR
	}

	sshPk, err := ssh.NewPublicKey(pk)
	if err != nil {
		log.Println("ssh pk err: " + err.Error())
	} else {
		log.Println(sshPk.Type() + " " + base64.StdEncoding.EncodeToString(sshPk.Marshal()))
	}

	templateIter := template
	modulus := pk.N.Bytes()
	eBytes := &bytes.Buffer{}
	err = binary.Write(eBytes, binary.BigEndian, int64(pk.E))
	if err != nil {
		log.Println("public exponent binary encoding error: " + err.Error())
		return C.CKR_GENERAL_ERROR
	}
	e := eBytes.Bytes()
	for i := C.CK_ULONG(0); i < count; i++ {
		//	TODO: memory safety/leak: should we be allocating?
		switch (*templateIter)._type {
		case C.CKA_ID:
			(*templateIter).pValue = unsafe.Pointer(C.CBytes(PUBKEY_ID))
			(*templateIter).ulValueLen = C.ulong(len(PUBKEY_ID))
		case C.CKA_MODULUS:
			log.Println("CKA_MODULUS")
			(*templateIter).pValue = unsafe.Pointer(C.CBytes(modulus))
			(*templateIter).ulValueLen = C.ulong(len(modulus))
		case C.CKA_MODULUS_BITS:
			log.Println("MODULUS_BITS")
			*(*C.CK_ULONG)((*templateIter).pValue) = C.CK_ULONG(pk.N.BitLen())
		case C.CKA_PUBLIC_EXPONENT:
			log.Println("CKA_PUBLIC_EXPONENT")
			(*templateIter).pValue = unsafe.Pointer(C.CBytes(e))
			(*templateIter).ulValueLen = C.ulong(len(e))
		case C.CKA_KEY_TYPE:
			log.Println("CKA_KEY_TYPE")
			rsaKeyType := (*C.CK_KEY_TYPE)(C.malloc(C.size_t(unsafe.Sizeof(C.CKK_RSA))))
			*rsaKeyType = C.CKK_RSA
			(*templateIter).pValue = unsafe.Pointer(rsaKeyType)
			(*templateIter).ulValueLen = C.ulong(unsafe.Sizeof(*rsaKeyType))
		case C.CKA_SIGN:
			log.Println("CKA_SIGN")
			*(*C.CK_BBOOL)((*templateIter).pValue) = C.CK_TRUE
		}

		templateIter = C.CK_ATTRIBUTE_PTR(unsafe.Pointer(uintptr(unsafe.Pointer(templateIter)) + unsafe.Sizeof(*template)))
	}
	return C.CKR_OK
}

//export C_SignInit
func C_SignInit(session C.CK_SESSION_HANDLE, mechanism C.CK_MECHANISM_PTR, key C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Println("C_SignInit")
	log.Println(mechanism.mechanism)
	switch mechanism.mechanism {
	case C.CKM_RSA_PKCS:
		return C.CKR_OK
	case C.CKM_RSA_X_509:
		log.Println("CKM_RSA_X_509 not supported")
		return C.CKR_MECHANISM_INVALID
	default:
		return C.CKR_MECHANISM_INVALID
	}
	return C.CKR_OK
}

//export C_Sign
func C_Sign(session C.CK_SESSION_HANDLE,
	data C.CK_BYTE_PTR, dataLen C.ulong,
	signature C.CK_BYTE_PTR, signatureLen *C.ulong) C.CK_RV {
	log.Println("C_Sign")
	log.Println("in sigLen", *signatureLen, "dataLen", dataLen)
	if signature == nil {
		*signatureLen = 512
		return C.CKR_OK
	}
	if *signatureLen < 512 {
		return C.CKR_BUFFER_TOO_SMALL
	}
	message := C.GoBytes(unsafe.Pointer(data), C.int(dataLen))
	pkFingerprint := sha256.Sum256(staticMe.SSHWirePublicKey)
	sigBytes, err := sign(pkFingerprint[:], message)
	//sigBytes, err := rsa.SignPKCS1v15(rand.Reader, sk, crypto.Hash(0), message)
	if err != nil {
		log.Println("sig error: " + err.Error())
		return C.CKR_GENERAL_ERROR
	} else {
		log.Println("got sig of", len(sigBytes), "bytes")
		for _, b := range sigBytes {
			*signature = C.CK_BYTE(b)
			signature = C.CK_BYTE_PTR(unsafe.Pointer(uintptr(unsafe.Pointer(signature)) + 1))
		}
		*signatureLen = C.ulong(len(sigBytes))
		log.Println("set sig")
	}
	return C.CKR_OK
}

//export C_Finalize
func C_Finalize(reserved C.CK_VOID_PTR) C.CK_RV {
	return C.CKR_OK
}
func bytesToChar64(b []byte) [64]C.uchar {
	for len(b) < 64 {
		b = append(b, byte(0))
	}
	return [64]C.uchar{
		C.uchar(b[0]), C.uchar(b[1]), C.uchar(b[2]), C.uchar(b[3]),
		C.uchar(b[4]), C.uchar(b[5]), C.uchar(b[6]), C.uchar(b[7]),
		C.uchar(b[8]), C.uchar(b[9]), C.uchar(b[10]), C.uchar(b[11]),
		C.uchar(b[12]), C.uchar(b[13]), C.uchar(b[14]), C.uchar(b[15]),
		C.uchar(b[16]), C.uchar(b[17]), C.uchar(b[18]), C.uchar(b[19]),
		C.uchar(b[20]), C.uchar(b[21]), C.uchar(b[22]), C.uchar(b[23]),
		C.uchar(b[24]), C.uchar(b[25]), C.uchar(b[26]), C.uchar(b[27]),
		C.uchar(b[28]), C.uchar(b[29]), C.uchar(b[30]), C.uchar(b[31]),
		C.uchar(b[32]), C.uchar(b[33]), C.uchar(b[34]), C.uchar(b[35]),
		C.uchar(b[36]), C.uchar(b[37]), C.uchar(b[38]), C.uchar(b[39]),
		C.uchar(b[40]), C.uchar(b[41]), C.uchar(b[42]), C.uchar(b[43]),
		C.uchar(b[44]), C.uchar(b[45]), C.uchar(b[46]), C.uchar(b[47]),
		C.uchar(b[48]), C.uchar(b[49]), C.uchar(b[50]), C.uchar(b[51]),
		C.uchar(b[52]), C.uchar(b[53]), C.uchar(b[54]), C.uchar(b[55]),
		C.uchar(b[56]), C.uchar(b[57]), C.uchar(b[58]), C.uchar(b[59]),
		C.uchar(b[60]), C.uchar(b[61]), C.uchar(b[62]), C.uchar(b[63]),
	}
}

func bytesToChar32(b []byte) [32]C.uchar {
	for len(b) < 32 {
		b = append(b, byte(0))
	}
	return [32]C.uchar{
		C.uchar(b[0]), C.uchar(b[1]), C.uchar(b[2]), C.uchar(b[3]),
		C.uchar(b[4]), C.uchar(b[5]), C.uchar(b[6]), C.uchar(b[7]),
		C.uchar(b[8]), C.uchar(b[9]), C.uchar(b[10]), C.uchar(b[11]),
		C.uchar(b[12]), C.uchar(b[13]), C.uchar(b[14]), C.uchar(b[15]),
		C.uchar(b[16]), C.uchar(b[17]), C.uchar(b[18]), C.uchar(b[19]),
		C.uchar(b[20]), C.uchar(b[21]), C.uchar(b[22]), C.uchar(b[23]),
		C.uchar(b[24]), C.uchar(b[25]), C.uchar(b[26]), C.uchar(b[27]),
		C.uchar(b[28]), C.uchar(b[29]), C.uchar(b[30]), C.uchar(b[31]),
	}
}

func bytesToChar16(b []byte) [16]C.uchar {
	for len(b) < 16 {
		b = append(b, byte(0))
	}
	return [16]C.uchar{
		C.uchar(b[0]), C.uchar(b[1]), C.uchar(b[2]), C.uchar(b[3]),
		C.uchar(b[4]), C.uchar(b[5]), C.uchar(b[6]), C.uchar(b[7]),
		C.uchar(b[8]), C.uchar(b[9]), C.uchar(b[10]), C.uchar(b[11]),
		C.uchar(b[12]), C.uchar(b[13]), C.uchar(b[14]), C.uchar(b[15]),
	}
}

func main() {}

var functions C.CK_FUNCTION_LIST = C.CK_FUNCTION_LIST{
	version: C.struct__CK_VERSION{
		major: 0,
		minor: 1,
	},
	C_Initialize:          C.CK_C_Initialize(C.C_Initialize),
	C_GetInfo:             C.CK_C_GetInfo(C.C_GetInfo),
	C_GetSlotList:         C.CK_C_GetSlotList(C.C_GetSlotList),
	C_GetSlotInfo:         C.CK_C_GetSlotInfo(C.C_GetSlotInfo),
	C_GetTokenInfo:        C.CK_C_GetTokenInfo(C.C_GetTokenInfo),
	C_OpenSession:         C.CK_C_OpenSession(C.C_OpenSession),
	C_CloseSession:        C.CK_C_CloseSession(C.C_CloseSession),
	C_FindObjectsInit:     C.CK_C_FindObjectsInit(C.C_FindObjectsInit),
	C_FindObjects:         C.CK_C_FindObjects(C.C_FindObjects),
	C_FindObjectsFinal:    C.CK_C_FindObjectsFinal(C.C_FindObjectsFinal),
	C_GetAttributeValue:   C.CK_C_GetAttributeValue(C.C_GetAttributeValue),
	C_SignInit:            C.CK_C_SignInit(C.C_SignInit),
	C_Sign:                C.CK_C_Sign(C.C_Sign),
	C_Finalize:            C.CK_C_Finalize(C.C_Finalize),
	C_GetMechanismList:    C.CK_C_GetMechanismList(C.C_GetMechanismList),
	C_GetMechanismInfo:    C.CK_C_GetMechanismInfo(C.C_GetMechanismInfo),
	C_InitToken:           C.CK_C_InitToken(C.C_InitToken),
	C_InitPIN:             C.CK_C_InitPIN(C.C_InitPIN),
	C_SetPIN:              C.CK_C_SetPIN(C.C_SetPIN),
	C_CloseAllSessions:    C.CK_C_CloseAllSessions(C.C_CloseAllSessions),
	C_GetSessionInfo:      C.CK_C_GetSessionInfo(C.C_GetSessionInfo),
	C_GetOperationState:   C.CK_C_GetOperationState(C.C_GetOperationState),
	C_SetOperationState:   C.CK_C_SetOperationState(C.C_SetOperationState),
	C_Login:               C.CK_C_Login(C.C_Login),
	C_Logout:              C.CK_C_Logout(C.C_Logout),
	C_CreateObject:        C.CK_C_CreateObject(C.C_CreateObject),
	C_CopyObject:          C.CK_C_CopyObject(C.C_CopyObject),
	C_DestroyObject:       C.CK_C_DestroyObject(C.C_DestroyObject),
	C_GetObjectSize:       C.CK_C_GetObjectSize(C.C_GetObjectSize),
	C_SetAttributeValue:   C.CK_C_SetAttributeValue(C.C_SetAttributeValue),
	C_EncryptInit:         C.CK_C_EncryptInit(C.C_EncryptInit),
	C_Encrypt:             C.CK_C_Encrypt(C.C_Encrypt),
	C_EncryptUpdate:       C.CK_C_EncryptUpdate(C.C_EncryptUpdate),
	C_EncryptFinal:        C.CK_C_EncryptFinal(C.C_EncryptFinal),
	C_DecryptInit:         C.CK_C_DecryptInit(C.C_DecryptInit),
	C_Decrypt:             C.CK_C_Decrypt(C.C_Decrypt),
	C_DecryptUpdate:       C.CK_C_DecryptUpdate(C.C_DecryptUpdate),
	C_DecryptFinal:        C.CK_C_DecryptFinal(C.C_DecryptFinal),
	C_DigestInit:          C.CK_C_DigestInit(C.C_DigestInit),
	C_Digest:              C.CK_C_Digest(C.C_Digest),
	C_DigestUpdate:        C.CK_C_DigestUpdate(C.C_DigestUpdate),
	C_DigestKey:           C.CK_C_DigestKey(C.C_DigestKey),
	C_DigestFinal:         C.CK_C_DigestFinal(C.C_DigestFinal),
	C_SignUpdate:          C.CK_C_SignUpdate(C.C_SignUpdate),
	C_SignFinal:           C.CK_C_SignFinal(C.C_SignFinal),
	C_SignRecoverInit:     C.CK_C_SignRecoverInit(C.C_SignRecoverInit),
	C_SignRecover:         C.CK_C_SignRecover(C.C_SignRecover),
	C_VerifyInit:          C.CK_C_VerifyInit(C.C_VerifyInit),
	C_Verify:              C.CK_C_Verify(C.C_Verify),
	C_VerifyUpdate:        C.CK_C_VerifyUpdate(C.C_VerifyUpdate),
	C_VerifyFinal:         C.CK_C_VerifyFinal(C.C_VerifyFinal),
	C_VerifyRecoverInit:   C.CK_C_VerifyRecoverInit(C.C_VerifyRecoverInit),
	C_VerifyRecover:       C.CK_C_VerifyRecover(C.C_VerifyRecover),
	C_DigestEncryptUpdate: C.CK_C_DigestEncryptUpdate(C.C_DigestEncryptUpdate),
	C_DecryptDigestUpdate: C.CK_C_DecryptDigestUpdate(C.C_DecryptDigestUpdate),
	C_SignEncryptUpdate:   C.CK_C_SignEncryptUpdate(C.C_SignEncryptUpdate),
	C_DecryptVerifyUpdate: C.CK_C_DecryptVerifyUpdate(C.C_DecryptVerifyUpdate),
	C_GenerateKey:         C.CK_C_GenerateKey(C.C_GenerateKey),
	C_GenerateKeyPair:     C.CK_C_GenerateKeyPair(C.C_GenerateKeyPair),
	C_WrapKey:             C.CK_C_WrapKey(C.C_WrapKey),
	C_UnwrapKey:           C.CK_C_UnwrapKey(C.C_UnwrapKey),
	C_DeriveKey:           C.CK_C_DeriveKey(C.C_DeriveKey),
	C_SeedRandom:          C.CK_C_SeedRandom(C.C_SeedRandom),
	C_GenerateRandom:      C.CK_C_GenerateRandom(C.C_GenerateRandom),
	C_GetFunctionStatus:   C.CK_C_GetFunctionStatus(C.C_GetFunctionStatus),
	C_CancelFunction:      C.CK_C_CancelFunction(C.C_CancelFunction),
	C_WaitForSlotEvent:    C.CK_C_WaitForSlotEvent(C.C_WaitForSlotEvent),
}