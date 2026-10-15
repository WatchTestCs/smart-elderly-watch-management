package com.example.careold.controller;

import com.example.careold.dao.PicShowDao;
import com.example.careold.domain.PicShow;
import java.util.LinkedHashMap;
import org.junit.jupiter.api.Test;
import java.util.Arrays;
import java.util.Collections;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class PicShowControllerTests {
    private LinkedHashMap params(Object files) {
        LinkedHashMap params=new LinkedHashMap();
        params.put("fileListName",files);
        return params;
    }
    @Test
    void invalidInputNeverDeletesExistingGallery() {
        PicShowController controller=new PicShowController();
        controller.picShowDao=mock(PicShowDao.class);
        for (LinkedHashMap input : Arrays.asList(null,new LinkedHashMap(),params(Collections.emptyList()),params(Arrays.asList("../bad.jpg")),params(Arrays.asList(1)))) {
            assertEquals("0000",controller.add(input).get("returnCode"));
        }
        verifyNoInteractions(controller.picShowDao);
    }
    @Test
    void oneInsertedRowReportsSuccess() {
        PicShowController controller=new PicShowController();
        controller.picShowDao=mock(PicShowDao.class);
        when(controller.picShowDao.save(any(PicShow.class))).thenReturn(1);
        assertEquals("1111",controller.add(params(Arrays.asList("care.jpg"))).get("returnCode"));
        verify(controller.picShowDao).delete(null);
        verify(controller.picShowDao).save(any(PicShow.class));
    }
    @Test
    void failedInsertRaisesAnErrorRatherThanReportingSuccess() {
        PicShowController controller=new PicShowController();
        controller.picShowDao=mock(PicShowDao.class);
        when(controller.picShowDao.save(any(PicShow.class))).thenReturn(1,0);
        assertThrows(IllegalStateException.class,()->controller.add(params(Arrays.asList("care.jpg","garden.jpg"))));
    }
}
