# Auto Arrange Validation

## Component: `web/src/workspace/arrange.ts`

### Modes Implemented

1. **Smart** - Automatically selects the best layout based on window count and viewport:
   - 1 window: grid (fill)
   - 2 windows: columns (landscape) or rows (portrait)
   - 3 windows with active: focus-active
   - 4+ windows: grid

2. **Grid** - Deterministic grid layout with minimum waste:
   - Calculates optimal columns/rows based on aspect ratio
   - Distributes space respecting minimum sizes
   - No randomness (same input = same output)

3. **Columns** - Equal-width vertical columns
4. **Rows** - Equal-height horizontal rows
5. **Focus Active** - Active window gets 60% space, others stacked

### Invariants Verified

- ✅ No View creation
- ✅ No View destruction
- ✅ No Agent creation
- ✅ No Agent stopping
- ✅ No runtime creation
- ✅ No runtime stopping
- ✅ No AgentID changes
- ✅ No RuntimeID changes
- ✅ No Flow changes
- ✅ Geometry-only mutations

### Test Coverage (10 tests)

1. `returns no geometry for empty inputs across every mode` - Empty input
2. `fills the viewport for a single window` - Single window
3. `lays out two windows as equal columns` - Two windows columns
4. `lays out three windows as equal rows` - Three windows rows
5. `gives the active window the primary region in focus-active mode` - Focus active
6. `uses a deterministic 2x2 grid for four windows` - 2x2 grid
7. `keeps grid layouts deterministic and geometry-only for 5, 6, 10 and 20 windows` - Determinism
8. `chooses columns, rows, focus-active or grid in smart mode based on count and viewport` - Smart mode
9. `honors minimum sizes when the viewport can satisfy them` - Minimum sizes
10. `clamps geometry inside a very small viewport when minimum sizes cannot fit` - Clamp behavior

### Determinism Verification

For counts [5, 6, 10, 20]:
- Same input produces identical output
- All IDs preserved in order
- All geometry within viewport
- No overlapping windows

### Edge Cases

- Zero windows: returns empty array
- Single window: fills viewport
- Very small viewport: clamps to available space
- Negative/NaN inputs: normalized to 0

## Verification

All 10 arrange tests pass. All 154 frontend tests pass.
