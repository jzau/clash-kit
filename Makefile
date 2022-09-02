build:
	@gomobile bind -o ./target/ClashKit.xcframework -target=ios,iossimulator -ldflags=-w ./clash
