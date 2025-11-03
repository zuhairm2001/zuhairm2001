import fs from "fs";
import { marked } from "marked";
import puppeteer from "puppeteer";
import {
  S3Client,
  ListBucketsCommand,
  ListObjectsCommand,
  GetObjectCommand,
  PutObjectCommand,
} from "@aws-sdk/client-s3";

const markdown = fs.readFileSync("resume.md", "utf-8");
const htmlContent = marked(markdown);
const css = fs.readFileSync("resume.css");

const fullHTML = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Styled Markdown PDF</title>
  <style>${css}</style>
</head>
<body class="page">
  ${htmlContent}
</body>
</html>
`;

(async () => {
  const browser = await puppeteer.launch();
  const page = await browser.newPage();
  await page.setContent(fullHTML, { waitUntil: "load" });
  await page.pdf({
    path: "output.pdf",
    format: "A4",
    printBackground: true,
  });
  await browser.close();
  console.log("PDF generated: output.pdf");
})();

const S3 = new S3Client({
  region: "auto",
  endpoint: Bun.env.S3_ENDPOINT!,
  credentials: {
    accessKeyId: Bun.env.S3_ACCESS_KEY!,
    secretAccessKey: Bun.env.S3_SECRET_KEY!,
  },
});

await S3.send(new PutObjectCommand({
  Bucket: "personal-bucket",
  Key: "resume.pdf",
  Body: fs.readFileSync("output.pdf"),
  ContentType: "application/pdf",
}));

console.log(
  await S3.send(new ListObjectsCommand({ Bucket: "personal-bucket" })),
);
