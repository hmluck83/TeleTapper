var AESIGEDecryptAddr = "0x6E6C860";
var AESIGEEncryptAddr = "0x152D1A0";

var moduleName = "Telegram";
var telegramModule = Process.getModuleByName(moduleName);
var baseAddr = telegramModule.base;

var decryptFuncAddr = baseAddr.add(AESIGEDecryptAddr);
var encryptFuncAddr = baseAddr.add(AESIGEEncryptAddr);

// Attach to AESIGE encrypt function
try {
  Interceptor.attach(encryptFuncAddr, {
    onEnter: function (args) {
      this.msg = args[0];
      this.msgLen = args[2].toInt32();

      // 암호화 전 원본 데이터를 Go로 전송
      var data = this.msg.readByteArray(this.msgLen);
      send({
        direction: "send",
        data: Array.from(new Uint8Array(data)),
      });
    },
  });
} catch (e) {
  console.log("Error attaching to encrypt function: " + e.message);
}

// Attach to AESIGE decrypt function
try {
  Interceptor.attach(decryptFuncAddr, {
    onEnter: function (args) {
      this.msg = args[1];
      this.msgLen = args[2].toInt32();
    },
    onLeave: function (retval) {
      // 복호화 후 원본 데이터를 Go로 전송
      var data = this.msg.readByteArray(this.msgLen);
      send({
        direction: "receive",
        data: Array.from(new Uint8Array(data)),
      });
    },
  });
} catch (e) {
  console.log("Error attaching to decrypt function: " + e.message);
}
