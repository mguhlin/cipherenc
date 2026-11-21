#!/usr/bin/env python3
"""Optional maintainer check: build a format fixture with system libargon2/libsodium.

Neither library is used or required by the application. Run on a system with both
libraries installed; the saved fixture is tested by Go on every platform.
"""
import base64
import ctypes as c
import ctypes.util
import json
from pathlib import Path

argon = c.CDLL(ctypes.util.find_library('argon2'))
sodium = c.CDLL(ctypes.util.find_library('sodium'))
password = b'Synthetic independent passphrase!'
header = b'CENCX01!' + bytes(range(16)) + bytes(range(24))
plain = json.dumps({'name':'independent.txt','text':True,'data':base64.b64encode(b'Independent libsodium fixture').decode()},separators=(',',':')).encode()
key = c.create_string_buffer(32)
argon.argon2id_hash_raw.argtypes = [c.c_uint32,c.c_uint32,c.c_uint32,c.c_void_p,c.c_size_t,c.c_void_p,c.c_size_t,c.c_void_p,c.c_size_t]
assert argon.argon2id_hash_raw(3,65536,1,password,len(password),header[8:24],16,key,32) == 0
encrypted = c.create_string_buffer(len(plain)+16)
length = c.c_ulonglong()
sodium.crypto_aead_xchacha20poly1305_ietf_encrypt.argtypes = [c.c_void_p,c.POINTER(c.c_ulonglong),c.c_void_p,c.c_ulonglong,c.c_void_p,c.c_ulonglong,c.c_void_p,c.c_void_p,c.c_void_p]
assert sodium.crypto_aead_xchacha20poly1305_ietf_encrypt(encrypted,c.byref(length),plain,len(plain),header,len(header),None,header[24:48],key) == 0
out = {'password':password.decode(),'package':base64.b64encode(header+encrypted.raw[:length.value]).decode(),'data':'Independent libsodium fixture'}
destination = Path(__file__).resolve().parents[1] / 'tests/independent-xchacha.json'
destination.write_text(json.dumps(out,indent=2)+'\n')
print('Wrote independent Argon2id + XChaCha20-Poly1305 fixture.')
