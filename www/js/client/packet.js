define(["underscore"], function (_) {
    var Packet = function (type, id, msg, payload) {
        this.type       = type      ? type : "";
        this.id         = id        ? id : "";
        this.msg        = msg       ? msg : "";
        this.payload    = payload   ? payload : "";
        return this;
    };

    Packet.Type = {
        PT_DISCONNECT: 0,
        PT_CONNECT: 1,
        PT_HEARTBEAT: 2,
        PT_MESSAGE: 3,
        PT_JSON: 4,
        PT_EVENT: 5,
        PT_ACK: 6,
        PT_ERROR: 7,
        PT_NOOP: 8,
        PT_SIZE: 9
    };

    Packet.Decode = function(rawPacket) {
        var parts = rawPacket.split(":", 4);

        if (parts.length !== 4) {
            throw "InvalidPacketError " + rawPacket;
        }

        var packet = new Packet();
        packet.type       = parts[0];

        if (_.isNaN(parseInt(packet.type, 10))) {
            throw "InvalidPacketTypeError " + rawPacket;
        }

        packet.id  = parts[1];
        packet.msg = parts[2];

        var i = rawPacket.indexOf(":") + 1;
        i = rawPacket.indexOf(":", i) + 1;
        i = rawPacket.indexOf(":", i) + 1;

        packet.payload = rawPacket.substr(i);

        return packet;
    };

    var packetProperties = [
        "type",
        "id",
        "msg",
        "payload"
    ];

    Packet.Encode = function(packet) {
        // Guard against undefined Properties
        _.each(packetProperties, function(prop) {
            if (_.isUndefined(packet[prop])) {
                packet[prop] = "";
            }
        });

        return packet.type + ":" + packet.id + ":" + packet.msg + ":" + packet.payload;
    };

    Packet.JSON = function(msg, obj) {
        return new Packet(Packet.Type.PT_JSON, "", msg, JSON.stringify(obj));
    };

    Packet.prototype.Encode = function() {
        return Packet.Encode(this);
    };

    return Packet;
});
