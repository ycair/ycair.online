/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export', // 讓 Next.js 產生靜態檔案
  images: {
    unoptimized: true, // Tauri 環境不支援動態圖片優化
  },
};

export default nextConfig;