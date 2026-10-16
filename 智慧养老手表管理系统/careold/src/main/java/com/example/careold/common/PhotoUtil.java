package com.example.careold.common;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.web.multipart.MultipartFile;

import java.io.*;
import java.util.HashMap;
import java.util.Map;



public class PhotoUtil {

    private PhotoUtil(){}

    private static final Logger LOG = LoggerFactory.getLogger(PhotoUtil.class);
    public static Map<String,String> saveFile(MultipartFile filedata, String path) {
        Map<String,String> result = new HashMap<>();
        result.put(DefineData.ERROR_CODE, DefineData.PHOTO_SAVE_FAILED);
        result.put(DefineData.ERROR_MESSAGE, "请选择不超过5MB的有效 JPG、PNG 或 BMP 图片");
        if (filedata == null || filedata.isEmpty() || filedata.getSize() > 5 * 1024 * 1024) return result;
        java.nio.file.Path temporary = null;
        try (InputStream input = filedata.getInputStream();
             javax.imageio.stream.ImageInputStream images = javax.imageio.ImageIO.createImageInputStream(input)) {
            if (images == null) return result;
            var readers = javax.imageio.ImageIO.getImageReaders(images);
            if (!readers.hasNext()) return result;
            var reader = readers.next();
            try {
                String format = reader.getFormatName().toLowerCase(java.util.Locale.ROOT);
                if (!java.util.Set.of("jpeg", "jpg", "png", "bmp").contains(format)) return result;
                reader.setInput(images, true, true);
                int width = reader.getWidth(0), height = reader.getHeight(0);
                if (width < 1 || height < 1 || width > 8192 || height > 8192 || (long) width * height > 12000000) return result;
                var image = reader.read(0);
                if (image == null) return result;
                String extension = format.equals("jpeg") || format.equals("jpg") ? "jpg" : "png";
                String name = java.util.UUID.randomUUID() + "." + extension;
                java.nio.file.Path root = java.nio.file.Paths.get(path).toAbsolutePath().normalize();
                java.nio.file.Files.createDirectories(root);
                root = root.toRealPath();
                temporary = java.nio.file.Files.createTempFile(root, "image-", ".tmp");
                // 重编码实际像素，不保留原始文件名、附加脚本或 EXIF 元数据。
                if (!javax.imageio.ImageIO.write(image, extension, temporary.toFile())) return result;
                java.nio.file.Files.move(temporary, root.resolve(name));
                temporary = null;
                result.put(DefineData.ERROR_CODE, DefineData.PHOTO_SAVE_SUCCESS);
                result.put("filename", name);
                result.put(DefineData.ERROR_MESSAGE, "图片已保存");
            } finally { reader.dispose(); }
        } catch (IOException | IllegalArgumentException e) {
            LOG.warn("图片未能保存: {}", e.getClass().getSimpleName());
        } finally {
            if (temporary != null) try { java.nio.file.Files.deleteIfExists(temporary); } catch (IOException e) { LOG.warn("临时图片清理失败"); }
        }
        return result;
    }

}
