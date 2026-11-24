# Moonwing for Compás

Moonwing is a silent arcade game for the Compás external-app launcher. Its
flight and scoring follow the supplied `flappybird.c`, including the original
floating-point integration, gates, and collision order. Its
visuals come from the Moonwing artwork embedded in the executable.

An asset-free loading screen appears as soon as the display is ready, before
input discovery, score loading, and artwork decoding. It switches straight
to the menu when initialization finishes, with no minimum display time.

Simulation advances in fixed 16.667 ms steps, retaining fractional elapsed
time and limiting catch-up work after a slow frame. Presentation follows
blocking display refreshes, with a deadline limiter for drivers that do not
wait on page flips. Long stalls pause a run instead of advancing it unseen.

## Install

1. Unzip `moonwing-compas.zip`.
2. Copy the `moonwing` folder into `.apps` at the root of the microSD card.
   Create `.apps` if needed. The folder contains the `moonwing` executable,
   Compás's required `app.json` manifest, and the `icon.png` launcher icon.
3. On the player, open **More → External Apps → Moonwing** and approve
   the launch.

Tap **Fly** to start and tap to flap. Play/Pause or Volume Up also flaps.
After a crash, tap **Fly** on the score screen to start another run. Tap **×**
in the top-right corner or press Power to return to Compás.

The app keeps the highest completed score in
`.apps-data/moonwing/best-score.txt` on the microSD card.

## Artwork handoff

Share `moonwing-artwork.zip` with the artist. Save revised PNGs under
`artwork/` in the source directory with the corresponding filenames and
capitalization. Keep the existing canvas sizes for the most predictable
layout. Transparent pixels are supported.

Bitmap labels and buttons use integer pixel scaling so their lettering keeps
even strokes. The same viewport transform places the scene and its touch
targets on the panel.

The raster keeps two horizontal columns per simulation unit. This preserves
one-device-pixel background movement on the 480px panels without changing
parallax speed, sprite proportions, or touch coordinates. Artwork is widened
once at load time; the native RGB565 presentation path copies complete rows.
Bird rotation uses a static 0.1-degree coefficient table and incremental
integer sampling; the simulation's angle and flight integration are unchanged.

Backgrounds keep the same pixel scale as the sprites and are anchored above
the ground. Shorter panels crop sky from the top; taller panels reveal more
sky rather than stretching the scenery. The bottom 56 artwork rows are
covered by the separate ground sprite.
On `score-board.png`, keep space for the live numbers below the SCORE and
BEST labels. Keep the bird centered within each of its three animation frames.
`menu.png`, `sound-on.png`, and `sound-off.png` are reserved artwork.

The launcher icon is `icon.png` in the source directory, declared by the
`icon` field in `app.json`. Keep it square when replacing it. It is copied
into both the install folder and the source package by `make package`.

Run `make package` after copying the revised PNGs to embed them in a new
executable. Install the rebuilt `moonwing` folder to preview the revision
on the player.

## Build

With Go 1.21 or newer, Make, and zip installed, run `make package` in the
source directory. It builds a static Linux MIPS32 little-endian executable
with software floating point and writes `dist/moonwing-compas.zip`. The
archive includes a separate `moonwing-source` folder for rebuilding. To build
the app folder by itself, run `make`. Run `make test` to validate rendering,
touch routing, and the simulation against the C reference trace.

The app uses the Compás external-app protocol v1. The 480×720 and 480×800
playfields match the supplied code. Smaller screens use the same scale on
both axes, including text and touch targets. Unusually short panels get
side margins to keep the starting bird visible without distorting the scene.
R1's `hyn_ts` touch coordinates are treated as panel pixels, matching the
firmware, instead of scaling them against its advertised sensor range.

[Compás external-app guide](https://compasplayer.org/apps.html)
