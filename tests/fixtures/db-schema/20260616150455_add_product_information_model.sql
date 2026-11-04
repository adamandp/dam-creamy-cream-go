-- CreateTable
CREATE TABLE "product_information" (
    "id" UUID NOT NULL,
    "product_id" UUID NOT NULL,
    "highlights" VARCHAR(255) NOT NULL,
    "serving_suggestions" VARCHAR(255) NOT NULL,
    "product_details" VARCHAR(255) NOT NULL,
    "why_choose" VARCHAR(255) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "product_information_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "product_information_product_id_key" ON "product_information"("product_id");

-- AddForeignKey
ALTER TABLE "product_information" ADD CONSTRAINT "product_information_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
