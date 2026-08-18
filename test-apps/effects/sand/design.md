# Sand Particle

```c++
struct particle_t
{
    u8 m_material;    // material type
    u8 m_density;     // density/intensity depending on the material
    u16 m_velocity_x; // 8.8 fixed point
    u16 m_velocity_y; // 8.8 fixed point
    u16 m_lifetime;   // lifetime (unit = 1.0/60.0 seconds)
};
```

# Action List

What if a piece of wood is acted on by fire, then we insert an 'action'?

Also, to me there seems to be a difference in the update frequency of certain things, like:

- gravity, should be done at the required frame-rate, e.g. 60hz or 30hz
- actions, should be done at a slower rate, e.g. 10hz, or even process the full list accross multiple frames, e.g. 1/10th of the list per frame, so that we don't have to process the full list every frame.

e.g.
```c++
struct action_t
{
    u8  m_action_type; // type of action (e.g. burning, melting, etc.)
    u8  m_duration;    // duration of the action in units as 1/32 seconds
    u16 m_particle;    // index of the particle that is being acted upon
};
```

Then we can have a list of actions that are currently affecting a particle, and we can update the particle's state based on the actions applied to it. This also means that we don't really need to scan the grid, we only need to process the 'action' list.

```c++
struct particle_t
{
    u8  m_material;    // material type
    u8  m_density;     // density/intensity depending on the material
    u16 m_action;      // index of action that is active for this particle
};
```

# Notes

What if we only use:

bit 7 = 'fire'.

bits 6-5 = 'state' (gas, liquid, solid)
Gas     = Gas                    = 00 (Rising physics)
Liquid  = Liquid                 = 01 (Flowing physics)
CSolid  = Cohesive Solid         = 10 (Cohesive blocks)
GSolid  = Granular Solid         = 11 (Granular blocks)

Then we really use the 'density' field to determine the viscosity for liquids, the strength left in an acid, the amount of wood left that is on fire before it turns into ember, and how much strength is left in an ember before it turns into nothing. How much water is left when slowly turning into steam, and how much oil is left if it is on fire. For smoke it is the density of the smoke, and smoke can spawn more smoke, and then the density of the smoke can determine how much more smoke it can spawn before it dissipates.

For Ice, density means how much strength is left in the ice before it turns into water. For stone, density means how much strength is left in the stone before it turns into dust. For sand, density means how much strength is left in the sand before it turns into dust. For salt, density means how much strength is left in the salt before it turns into nothing. For mud, density means how much strength is left in the mud before it turns into water, or when increasing in density it turning into ground. For metal, density means how much strength is left in the metal before it turns into dust. For wood, density means how much strength is left in the wood before it turns into nothing. For lava density means how much strength is left in the lava before it turns into stone. For ember, density means how much strength is left in the ember before it turns into nothing.

Then we can use the other 5 bits for material type, and we can have 32 different materials.

# 8-Bit Material Bitmask Schema

States:

Gas     = Gas                    = 000 (Rising physics)
CSolid  = Cohesive Solid         = 010 (Fixed blocks)
GSolid  = Granular Solid         = 011 (Physics blocks)
LLiquid = Low-Density Liquid     = 100
SLiquid = Standard Liquid        = 101
VLiquid = Viscous/Heavy Liquid   = 110

Bit 7:

1000 : Highly Energized / Reactive (Explosive, Combustible, etc.)

Elements:

- Air    = 0000
- Water  = 0001
- Acid   = 0010
- Oil    = 0011
- Stone  = 0100
- Metal  = 0101
- Wood   = 0110
- Sand   = 0111
- Earth  = 1000
- Salt   = 1001

| Material | Energized | State | Element | Physics Group | Sub-Type Identity |
|---|---|---:|---:|---|---|
| Empty     | `0` | Gas     |   Air  | Air / Void | Default Empty |
| Steam     | `0` | Gas     |  Water | Gases (Rising) | Condensing Vapor |
| Smoke     | `0` | Gas     |  Wood  | Gases (Rising) | Dissipating Aerosol |
| Smoke     | `0` | Gas     |  Oil   | Gases (Rising) | Dissipating Aerosol |
| Smoke     | `0` | Gas     |  Acid  | Gases (Rising) | Dissipating Aerosol |
| Glass     | `0` | CSolid  |  Sand  | Cohesive Solid | Brittle Transparent |
| Ice       | `0` | CSolid  |  Water | Cohesive Solid | Slippery Thermal |
| Stone     | `0` | CSolid  |  Stone | Cohesive Solid | Blast-Resistant |
| Dust      | `0` |  Gas    |  Earth | Gases (Rising) | Fine Particulate |
| Ground    | `0` | CSolid  |  Earth | Cohesive Solid | Absorbent Dirt |
| Mud       | `0` | VLiquid |  Earth | Viscous/Heavy Liquids | Thick Sludge |
| Metal     | `0` | CSolid  |  Metal | Cohesive Solid | Conductive Heavy |
| Wood      | `0` | CSolid  |  Wood  | Cohesive Solid | Structural Flammable |
| Sand      | `0` | GSolid  |  Sand  | Granular Solids | Granular Mineral |
| Salt      | `0` | GSolid  |  Salt  | Granular Solids | Soluble Crystal |
| Snow      | `0` | GSolid  | Water  | Granular Solids | Light Thermal Stack |
| Gunpowder | `1` | GSolid  |  Earth | Energy / Reactive | Explosive Combustible |
| Oil       | `0` | LLiquid |  Oil   | Low-Density Liquids | Flammable Fuel |
| Water     | `0` | SLiquid |  Water | Standard Liquids | Default Solvent |
| Acid      | `0` | SLiquid |  Acid  | Standard Liquids | Corrosive Reactive |
| Lava      | `0` | VLiquid |  Stone | Viscous/Heavy Liquids | Molten Thermal |
| Ember     | `1` | GSolid  |  Stone | Energy / Reactive | Stone Embers |
| Ember     | `1` | CSolid  |  Wood  | Energy / Reactive | Embers from Wood |
| Fire      | `1` | GSolid  |  Oil   | Energy / Reactive | Oil on Fire |
| Fire      | `1` | GSolid  |  Wood  | Energy / Reactive | Wood on Fire |
| Fire      | `1` |  Gas    |  Oil   | Energy / Reactive | Flames from Oil on Fire |
| Fire      | `1` |  Gas    |  Wood  | Energy / Reactive | Flames from Wood on Fire |
| Fire      | `1` |  Gas    |  Air   | Energy / Reactive | Flames |
