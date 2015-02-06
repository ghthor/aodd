define(["client/packet",
       "underscore",
       "lib/jasmine"
], function(Packet, _) {
    describe("Packet", function() {

        Packet.prototype.equals = function(packet) {
            return (packet.type     === this.type &&
                    packet.id       === this.id &&
                    packet.msg      === this.msg &&
                    packet.payload  === this.payload);
        };

        it("should marshal objects without all required properties", function() {
            var testPacket = {
                type: Packet.Type.PT_MESSAGE,
                msg: "message"
            };

            var packetStr = Packet.Encode(testPacket);
            expect(packetStr).toEqual("3::message:");
        });

        it("should error unmarshalling invalid packet strings into Packet Objects", function() {
            var testPackets = [
                "",
                ":",
                "::"
            ];

            _.each(testPackets, function(packet) {
                expect(function() {
                    Packet.Decode(packet);
                }).toThrow();
            });
        });

        it("should error unmarshalling packet\"s with invalid PacketTypes", function() {
            var testPackets = [
                ":::",
                "a:::"
            ];

            _.each(testPackets, function(packet) {
                expect(function() {
                    Packet.Decode(packet);
                }).toThrow();
            });
        });
    });
});
