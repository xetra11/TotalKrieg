# TotalKrieg

Rundenbasiertes Operationsspiel auf einer Hexkarte: Wehrmacht gegen Sowjetunion, Spieler gegen KI.
Prototyp in Go mit [Ebitengine](https://ebitengine.org). Die Regeln stehen in [docs/GDD.md](docs/GDD.md).

## Starten

Voraussetzung ist Go 1.26 oder neuer.

```bash
go run ./cmd/totalkrieg
```

Optionen:

| Option | Wirkung |
| --- | --- |
| `-seed 7` | Spielt eine bestimmte Karte, 0 wählt zufällig |
| `-faction wehrmacht` | Überspringt das Menü, auch `sowjetunion` |
| `-autoplay 18` | Spielt so viele KI-Züge vor dem Start |
| `-shot bild.png` | Schreibt ein Bild und beendet das Spiel |

## Bedienung

Eine Runde läuft in drei Phasen: Aufstellung, Bewegung, Angriff. Einkommen kommt zu Zugbeginn, Eroberungen zählen am Zugende.

- **Aufstellung:** Eigene Stadt anklicken, dann rechts eine Einheit wählen.
- **Bewegung:** Eigene Einheit anklicken, dann ein markiertes Feld.
- **Angriff:** Eigene Einheit anklicken, dann einen rot markierten Gegner.
- **Gliederung:** Einheit anklicken, dann **G** oder den Knopf rechts. Das Fenster zeigt den ganzen Verband mit Rufnamen und Kompanieführern.
- **Ausgliedern:** In der Bewegungsphase in der Gliederung auf „Ausgliedern“ klicken, dann ein markiertes Nachbarfeld wählen.
- **Eingliedern:** Teilverband neben den Stammverband ziehen, dann rechts auf „Eingliedern“ klicken.
- **Enter** oder der Knopf oben rechts geht zur nächsten Phase, **Esc** bricht ab oder hebt die Auswahl auf.

## Aufbau

| Pfad | Inhalt |
| --- | --- |
| `cmd/totalkrieg` | Programmstart |
| `internal/game` | Regeln, Kartenerzeugung und KI, ohne Grafik |
| `internal/symbols` | Symbolsets pro Fraktion als JSON |
| `internal/ui` | Oberfläche mit Ebitengine |
| `concept` | Konzeptgrafik und ihre Quelle |
| `docs` | GDD und Quellen |
| `tools` | Export der Konzeptgrafik |

## Tests

```bash
go test ./...
```

`TestFullMatches` lässt die KI 400 Partien gegen sich selbst spielen. Mit `go test ./internal/game -run TestFullMatches -v` zeigt der Test die Siegverteilung.
