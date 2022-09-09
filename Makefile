build:
	@gomobile bind -o ./target/ClashKit.xcframework -target=ios,iossimulator -ldflags=-w ./clash
	@gomobile bind -o ./target/ClashKit.aar -target=android -ldflags=-w ./clash
