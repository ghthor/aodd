define(["client/player",
       "client/sprite/human",
       "client/imageCache",
       "underscore",
       "lib/jasmine"
], function(Player, Human, ImageCache, _) {

    var stage = {
        width: 400,
        height: 300,
        grid: 15
    };

    var MockDirector = function(imageCache) {
        this.getImage = function(id) {
            return _.find(imageCache, function(image) {
                return id === image.id;
            });
        };
    };

    var MockScene = function(width, height) {
        this.width = width;
        this.height = height;
        this.addChild = jasmine.createSpy("addChild");
    };

    var MockWorld = function(scene) {
        this.scene = scene;
        this.grid = stage.grid;
    };
    MockWorld.prototype.move = function() {};

    var describeActors = function(ready) {
        // Async Setup
        var imageCache = new ImageCache();
        imageCache.on("complete", ready);
        imageCache.loadDefault();

        describe("actors", function() {
            var director, scene;

            beforeEach(function() {
                director = new MockDirector(imageCache.images);
                scene = new MockScene(stage.width, stage.height);
            });

            afterEach(function() {
                director = undefined;
                scene = undefined;
            });

            describe("player actor", function() {
                var playerEntity, player, world;

                beforeEach(function() {
                    playerEntity = {
                        id:          0,
                        name:        "PlayerEntity",
                        facing:      "north",
                        pathAction: null,
                        cell:        {x: 0, y: 0}
                    };
                    world = new MockWorld(scene);
                    player = new Player(null, world, playerEntity);
                });

                afterEach(function() {
                    world = undefined;
                    player = undefined;
                });

                it("must create an actor when it recieves the first update", function() {
                    // Actor wasn't created during constructor
                    expect(player.actor).not.toBeDefined();
                    expect(player.sprite).not.toBeDefined();

                    player.createActor = function() {
                        return { setAnimation: jasmine.createSpy("setAnimation") };
                    };
                    spyOn(player, "createActor").andCallThrough();

                    player.setHealthPercentage = function(){};

                    player.update(0, playerEntity);
                    expect(player.createActor).toHaveBeenCalled();
                    expect(world.scene.addChild).toHaveBeenCalled();
                    expect(player.actor.setAnimation).toHaveBeenCalled();
                    
                    var args = player.createActor.calls[0].args;
                    expect(playerEntity.name).toBe(args[0]);
                    // Positioning
                    expect(args[1]).toBe(stage.width/2);
                    expect(args[2]).toBe(stage.height/2);
                });

                it("must move the world container when the player recieves an update with a new pathAction", function() {
                    player.createActor = function() {
                        return { setAnimation: jasmine.createSpy("setAnimation") };
                    };

                    player.setHealthPercentage = function(){};

                    playerEntity.pathAction = {
                        orig: playerEntity.cell,
                        dest: {x: 0, y: 1},
                        start: 0,
                        end: 10
                    };

                    spyOn(world, "move");
                    player.update(0, playerEntity);

                    expect(world.move).toHaveBeenCalled();
                    expect(world.move.calls.length).toBe(1);
                });
            });

            describe("human actor", function() {
                var human, sprites;

                var MockSprite = function(frames) {
                    this.frames = frames;
                    spyOn(this, "setVisible").andCallThrough();
                };
                MockSprite.prototype = {
                    setVisible: function() {
                        return this;
                    },
                    resetAnimationTime: function() {
                        return this;
                    }
                };

                beforeEach(function() {
                    human = new Human("female");

                    human.sprites = sprites = {};
                    _.each(Human.animations, function(frames, animation) {
                        sprites[animation] = new MockSprite(frames);
                    });
                });

                afterEach(function() {
                    human = undefined;
                });

                it("sets animation to standing", function() {
                    expect(human.sprite).not.toBeDefined();
                    expect(human.sprites).toBeDefined();

                    var entity = { pathAction: null, facing: "north" };
                    human.setAnimation(entity);
                    expect(human.sprite).toBe(sprites.standNorth);

                    var setVisible = sprites.standNorth.setVisible;
                    expect(setVisible).toHaveBeenCalled();
                    expect(setVisible.calls[0].args[0]).toBe(true);

                    entity.facing = "east";
                    human.setAnimation(entity);
                    expect(setVisible.calls.length).toBe(2);
                    expect(setVisible.calls[1].args[0]).toBe(false);
                    expect(human.sprite).toBe(sprites.standEast);

                    setVisible = sprites.standEast.setVisible;
                    expect(setVisible).toHaveBeenCalled();
                    expect(setVisible.calls[0].args[0]).toBe(true);

                    entity.facing = "south";
                    human.setAnimation(entity);
                    expect(setVisible.calls.length).toBe(2);
                    expect(setVisible.calls[1].args[0]).toBe(false);
                    expect(human.sprite).toBe(sprites.standSouth);

                    setVisible = sprites.standSouth.setVisible;
                    expect(setVisible).toHaveBeenCalled();
                    expect(setVisible.calls[0].args[0]).toBe(true);

                    entity.facing = "west";
                    human.setAnimation(entity);
                    expect(setVisible.calls.length).toBe(2);
                    expect(setVisible.calls[1].args[0]).toBe(false);
                    expect(human.sprite).toBe(sprites.standWest);

                    setVisible = sprites.standWest.setVisible;
                    expect(setVisible).toHaveBeenCalled();
                    expect(setVisible.calls[0].args[0]).toBe(true);
                });

                it("sets animation to walking", function() {
                    expect(human.sprite).not.toBeDefined();
                    expect(human.sprites).toBeDefined();

                    var entity = { pathAction: {}, facing: "north" };
                    human.setAnimation(entity);
                    expect(human.sprite).toBe(sprites.walkNorth);

                    var setVisible = sprites.walkNorth.setVisible;
                    expect(setVisible).toHaveBeenCalled();
                    expect(setVisible.calls[0].args[0]).toBe(true);

                    entity.facing = "east";
                    human.setAnimation(entity);
                    expect(setVisible.calls.length).toBe(2);
                    expect(setVisible.calls[1].args[0]).toBe(false);
                    expect(human.sprite).toBe(sprites.walkEast);

                    setVisible = sprites.walkEast.setVisible;
                    expect(setVisible).toHaveBeenCalled();
                    expect(setVisible.calls[0].args[0]).toBe(true);

                    entity.facing = "south";
                    human.setAnimation(entity);
                    expect(setVisible.calls.length).toBe(2);
                    expect(setVisible.calls[1].args[0]).toBe(false);
                    expect(human.sprite).toBe(sprites.walkSouth);

                    setVisible = sprites.walkSouth.setVisible;
                    expect(setVisible).toHaveBeenCalled();
                    expect(setVisible.calls[0].args[0]).toBe(true);

                    entity.facing = "west";
                    human.setAnimation(entity);
                    expect(setVisible.calls.length).toBe(2);
                    expect(setVisible.calls[1].args[0]).toBe(false);
                    expect(human.sprite).toBe(sprites.walkWest);

                    setVisible = sprites.walkWest.setVisible;
                    expect(setVisible).toHaveBeenCalled();
                    expect(setVisible.calls[0].args[0]).toBe(true);
                });
            });
        });
    };

    return describeActors;
});
