/*
  Warnings:

  - Changed the type of `highlights` on the `product_information` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Changed the type of `serving_suggestions` on the `product_information` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Changed the type of `product_details` on the `product_information` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.

*/
-- AlterTable
ALTER TABLE "product_information" DROP COLUMN "highlights",
ADD COLUMN     "highlights" JSONB NOT NULL,
DROP COLUMN "serving_suggestions",
ADD COLUMN     "serving_suggestions" JSONB NOT NULL,
DROP COLUMN "product_details",
ADD COLUMN     "product_details" JSONB NOT NULL,
ALTER COLUMN "why_choose" SET DATA TYPE TEXT;
