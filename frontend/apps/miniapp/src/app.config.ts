export default defineAppConfig({
  pages: [
    "pages/home/index",
    "pages/location/index",
    "pages/sites/index",
    "pages/site-detail/index",
    "pages/locker-selection/index",
    "pages/order-confirm/index",
    "pages/payment/index"
  ],
  window: {
    navigationBarTitleText: "智能快递柜",
    navigationBarBackgroundColor: "#FFFFFF",
    navigationBarTextStyle: "black",
    backgroundColor: "#F7F9FC"
  },
  permission: {
    "scope.userLocation": {
      desc: "用于查找附近的智能寄存柜网点"
    }
  },
  requiredPrivateInfos: ["getLocation"],
  tabBar: {
    color: "#64748B",
    selectedColor: "#1769E0",
    backgroundColor: "#FFFFFF",
    list: [
      { pagePath: "pages/home/index", text: "首页" },
      { pagePath: "pages/sites/index", text: "网点" }
    ]
  }
});
