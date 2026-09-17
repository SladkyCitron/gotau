<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-white.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/logo-black.svg">
    <img src="assets/logo-black.svg" alt="GoTAU Logo" width="600">
  </picture>
</p>

[![CI (Go)](https://github.com/SladkyCitron/gotau/actions/workflows/ci.yml/badge.svg)](https://github.com/SladkyCitron/gotau/actions/workflows/ci.yml) [![GitHub license](https://img.shields.io/github/license/SladkyCitron/gotau)](LICENSE) ![Made in Slovakia](https://raw.githubusercontent.com/pedromxavier/flag-badges/refs/heads/main/badges/SK.svg)

**GoTAU** (pronounced _Go UTAU_) is a work-in-progress modern UTAU-compatible singing voice synthesizer written in Go.
It's designed to be fast, modern, modular, and easy to use, while staying backwards compatible with the existing UTAU ecosystem.

Built with ❤️ for the vocal synth community.

## 🚧 Project Status

GoTAU is currently in early development.

Core systems are still being implemented, and many features are incomplete.
Expect bugs and breaking changes.

## ✨ Features

- Fast and efficient synthesis engine
- Cross-platform support
- Backwards compatibility with existing UST files and UTAU voicebanks
- Modular architecture for easy extension

### Planned Features

- Built-in resampler / concatenator
- Plugin support
- GUI

## ⚠️ Known Limitations

- No built-in resampler yet, users have to use an external one for now (I recommend [straycat-rs](https://github.com/UtaUtaUtau/straycat-rs))
- Only supports CV and VCV voicebanks for now

## 🐹 Why Go?

Most vocal synth frontends are written in C# or Java, and the underlying synthesis engines are often written in lower-level languages like C++.
GoTAU explores what a singing voice synthesizer would look like if it were mostly written in Go, leveraging Go's modern tooling, simplicity, concurrency model, and performance.

Also, I just really like Go and singing voice synthesis tech and wanted to see if I could build my own vocal synth with it!

## 📖 The name

The name "GoTAU" is a portmanteau of "Go" and "UTAU", pronounced "Go UTAU".

## ⚖️ License

The GoTAU source code is licensed under the MIT License (see [LICENSE](LICENSE)).

The GoTAU [logo](assets/logo-master.svg), [icon](assets/icon-master.svg), and other branding assets by SladkyCitron are licensed under the [Creative Commons Attribution-NoDerivatives 4.0 International License](https://creativecommons.org/licenses/by-nd/4.0/).

## ❤️ Acknowledgements

- UTAU by Ameya/Ayame (飴屋／菖蒲)
- [OpenUtau](https://github.com/openutau/OpenUtau)
- The Vocaloid & UTAU community
