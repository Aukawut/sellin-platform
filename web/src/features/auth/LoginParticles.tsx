/** Decorative background only; never intercepts login interactions. */
const circuits = [
  'M0 180H180L290 290H450L560 180H880L990 290H1150L1260 180H1440',
  'M0 690H210L330 570H470L590 690H850L970 570H1110L1230 690H1440',
  'M80 0V120L220 260V440L100 560V900',
  'M1360 0V120L1220 260V440L1340 560V900',
  'M300 0V100L400 200V350L300 450V710L440 850V900',
  'M1140 0V100L1040 200V350L1140 450V710L1000 850V900',
  'M0 400H110L190 480H380L480 380H960L1060 480H1250L1330 400H1440',
  'M0 820H260L360 720H520L620 820H820L920 720H1080L1180 820H1440',
]

export function LoginParticles() {
  return (
    <div className="login-particles" aria-hidden="true">
      <div className="login-particles-glow" />
      <svg viewBox="0 0 1440 900" preserveAspectRatio="xMidYMid slice" fill="none" focusable="false">
        <g className="login-circuit-lines">
          {circuits.map((d) => <path key={d} d={d} />)}
        </g>
        <g className="login-circuit-signals">
          {circuits.map((d) => <path key={d} d={d} />)}
        </g>
        <g className="login-particle-nodes">
          {[
            [180, 180], [290, 290], [450, 290], [560, 180], [880, 180],
            [990, 290], [1150, 290], [1260, 180], [210, 690], [330, 570],
            [470, 570], [590, 690], [850, 690], [970, 570], [1110, 570],
            [1230, 690], [220, 440], [1220, 440], [300, 710], [1140, 710],
            [110, 400], [190, 480], [1250, 480], [1330, 400], [620, 820], [820, 820],
          ].map(([cx, cy]) => <circle key={`${cx}-${cy}`} cx={cx} cy={cy} r="3" />)}
        </g>
        <g className="login-particle-dust">
          {Array.from({ length: 64 }, (_, i) => (
            <circle key={i} cx={24 + (i * 137) % 1392} cy={20 + (i * 97) % 860} r={i % 3 === 0 ? 2 : 1} />
          ))}
        </g>
      </svg>
    </div>
  )
}
