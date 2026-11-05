(function () {
  if (!window.mermaid) return;

  window.mermaid.initialize({
    startOnLoad: false,
    securityLevel: 'strict',
    theme: 'base',
    fontFamily: 'Inter, "PingFang SC", "Microsoft YaHei", sans-serif',
    themeVariables: {
      primaryColor: '#edf8f0',
      primaryTextColor: '#173b2c',
      primaryBorderColor: '#47976d',
      secondaryColor: '#f5f1e8',
      tertiaryColor: '#fff7e5',
      fontSize: '16px',
      lineColor: '#557065',
      textColor: '#243a31',
      actorBkg: '#edf8f0',
      actorBorder: '#47976d',
      actorTextColor: '#173b2c',
      signalColor: '#557065',
      signalTextColor: '#243a31',
      noteBkgColor: '#fff7e5',
      noteBorderColor: '#d8b56c',
      noteTextColor: '#59451d'
    },
    flowchart: {
      useMaxWidth: true,
      curve: 'basis',
      htmlLabels: false
    },
    sequence: {
      useMaxWidth: true,
      mirrorActors: false,
      wrap: true
    }
  });

  const diagrams = [...document.querySelectorAll('.mermaid')];
  if (!diagrams.length) return;

  window.mermaid.run({ nodes: diagrams }).catch((error) => {
    console.error('Mermaid diagram render failed:', error);
  });
})();
