figma.showUI(__html__, { width: 420, height: 620 });

figma.ui.onmessage = (message: { type?: string }) => {
  if (message.type === "close-plugin") {
    figma.closePlugin();
  }
};
