define(["underscore",
       "CAAT"
], function(_) {
    var animations = {
        standNorth: [1],
        standEast:  [4],
        standSouth: [7],
        standWest:  [10],
        walkNorth:  [0,1,2,1],
        walkEast:   [3,4,5,4],
        walkSouth:  [6,7,8,7],
        walkWest:   [9,10,11,10]
    };

    var Human = function(gender) {
        this.gender = gender;
    };

    Human.makeImage = function(director, gender) {
        return new CAAT.SpriteImage().initialize(director.getImage(gender), 4, 3);
    };

    var makeSprites = function(img, posX, posY) {
        var sprites = {};
        _.each(animations, function(frames, animation) {
            sprites[animation] = new CAAT.Actor().
                setBackgroundImage(img.getRef(), true).
                setAnimationImageIndex(frames).
                setChangeFPS(300).
                setVisible(false).
                setPositionAnchored(posX, posY, 0.5, 0.5);
        });
        this.sprites = sprites;
        return this;
    };

    var setAnimation = function(entity) {
        var sprite, sprites = this.sprites;

        if (!_.isNull(entity.pathAction)) {
            // Walking
            switch (entity.facing) {
            case "north":
                sprite = sprites.walkNorth;
                break;
            case "east":
                sprite = sprites.walkEast;
                break;
            case "south":
                sprite = sprites.walkSouth;
                break;
            case "west":
                sprite = sprites.walkWest;
                break;
            default:
                throw "walking with unknown facing: " + entity.facing;
            }
        } else {
            // Standing
            switch (entity.facing) {
            case "north":
                sprite = sprites.standNorth;
                break;
            case "east":
                sprite = sprites.standEast;
                break;
            case "south":
                sprite = sprites.standSouth;
                break;
            case "west":
                sprite = sprites.standWest;
                break;
            default:
                throw "standing with unknown facing: " + entity.facing;
            }
        }

        if (sprite !== this.sprite) {
            if (!_.isUndefined(this.sprite)) {
                this.sprite.setVisible(false);
            }
            sprite.setVisible(true).resetAnimationTime();
            this.sprite = sprite;
        }
    };

    Human.animations = animations;

    Human.prototype = {
        makeSprites: makeSprites,
        setAnimation: setAnimation
    };

    return Human;
});
