
@protoc.exe --plugin=protoc-gen-go=%GOPATH%/bin/protoc-gen-go.exe --proto_path=. --go_out=.  ./github.com/3zheng/railgun/protodefine/mytype/types.proto
@protoc.exe --plugin=protoc-gen-go=%GOPATH%/bin/protoc-gen-go.exe --proto_path=. --go_out=.  ./github.com/3zheng/railgun/protodefine/tcpnet/tcp.proto
@protoc.exe --plugin=protoc-gen-go=%GOPATH%/bin/protoc-gen-go.exe --proto_path=. --go_out=.  ./github.com/3zheng/railgun/protodefine/gate/gate.proto
@protoc.exe --plugin=protoc-gen-go=%GOPATH%/bin/protoc-gen-go.exe --proto_path=. --go_out=.  ./github.com/3zheng/railgun/protodefine/router/router.proto
@protoc.exe --plugin=protoc-gen-go=%GOPATH%/bin/protoc-gen-go.exe --proto_path=. --go_out=.  ./github.com/3zheng/railgun/protodefine/AppFrame/AppFrame.proto
@protoc.exe --plugin=protoc-gen-go=%GOPATH%/bin/protoc-gen-go.exe --proto_path=. --go_out=.  ./github.com/3zheng/railgun/protodefine/client/client.proto

pause

@rem d:\vs2013\run\win32_release\protoc.exe --lua_out=.. logger/logger.proto
@rem d:\vs2013\run\win32_release\protoc.exe --cpp_out=.. gateclient/gateclient.proto
@rem d:\vs2013\run\win32_release\protoc.exe --go_out=.. gate/gate.proto